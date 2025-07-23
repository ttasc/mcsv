package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

const (
    writeWait           = 10 * time.Second      // Time allowed to write a message to the peer.
    maxMessageSize      = 8192                  // Maximum message size allowed from peer.
    pongWait            = 60 * time.Second      // Time allowed to read the next pong message from the peer.
    pingPeriod          = (pongWait * 9) / 10   // Send pings to peer with this period. Must be less than pongWait.
    closeGracePeriod    = 10 * time.Second      // Time to wait before force close on connection.
)

//go:embed web
var fs embed.FS

type WebServer struct {
    mcsv   *MinecraftServer
    Server *http.Server
}

func NewWebServer(config Webconfig, mcsv *MinecraftServer) WebServer {
    server := WebServer{
        mcsv:   mcsv,
    }

    // Declare Server config
    server.Server = &http.Server{
        Addr:         fmt.Sprintf(":%d", config.Port),
        Handler:      server.registerHandlers(),
        IdleTimeout:  time.Minute,
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 30 * time.Second,
    }

    return server
}

func (s *WebServer) GracefulShutdown(done chan bool) {
    // Create context that listens for the interrupt signal from the OS.
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    // Listen for the interrupt signal.
    <-ctx.Done()

    log.Println("shutting down gracefully, press Ctrl+C again to force")

    // The context is used to inform the server it has 5 seconds to finish
    // the request it is currently handling
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := s.Server.Shutdown(ctx); err != nil {
        log.Printf("Server forced to shutdown with error: %v", err)
    }

    log.Println("Server exiting")

    // Notify the main goroutine that the shutdown is complete
    done <- true
}

func (s *WebServer) registerHandlers() http.Handler {
    mux := http.NewServeMux()
    s.registerRoutes(mux) // Register routes
    return s.logMiddleware(s.corsMiddleware(mux)) // Wrap the mux with middleware
}

func (s *WebServer) corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Set CORS headers
        w.Header().Set("Access-Control-Allow-Origin", "*") // Replace "*" with specific origins if needed
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
        w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
        w.Header().Set("Access-Control-Allow-Credentials", "false") // Set to "true" if credentials are required

        // Handle preflight OPTIONS requests
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusNoContent)
            return
        }

        // Proceed with the next handler
        next.ServeHTTP(w, r)
    })
}

func (s *WebServer) logMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("Received request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
        next.ServeHTTP(w, r)
    })
}

func (s *WebServer) registerRoutes(mux *http.ServeMux) {
    // Static
    mux.Handle("/web/icon.png", http.FileServer(http.FS(fs)))

    // Http
    mux.HandleFunc("/",         s.dashboard)
    mux.HandleFunc("/start",    s.start)
    mux.HandleFunc("/stop",     s.stop)
    // mux.HandleFunc("/backup",   s.backup)
    // mux.HandleFunc("/players",  s.players)

    // WebSocket
    mux.HandleFunc("/console",  s.console)
}

func (s *WebServer) dashboard(w http.ResponseWriter, r *http.Request) {
    tmpl, err := template.ParseFS(fs, "web/index.html")
    if err != nil {
        log.Println("ERROR parsing template:", err)
        return
    }
    tmpl.Execute(w, nil)
}

func (s *WebServer) start(w http.ResponseWriter, r *http.Request) {
    newWorld, err := strconv.ParseBool(r.URL.Query().Get("newworld"))
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    if newWorld {
        if err = s.mcsv.RemoveOldWorld(); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
    }
    err = s.mcsv.StartMC(true)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func (s *WebServer) stop(w http.ResponseWriter, r *http.Request) {
    err := s.mcsv.StopMC()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func (s *WebServer) console(w http.ResponseWriter, r *http.Request) {
    if !IsMCRunningBackground(s.mcsv.DataPath) {
        w.Write([]byte("Minecraft is not running"))
        return
    }

    var upgrader = websocket.Upgrader{
        CheckOrigin : func(r *http.Request) bool { return true },
    }
    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    stdoutDone := make(chan struct{})
    go pumpStdout(ws, s.mcsv.FileO, stdoutDone)
    go ping(ws, stdoutDone)
    pumpStdin(ws, s.mcsv.FileI)
}

func pumpStdin(ws *websocket.Conn, FileI string) {
    defer ws.Close()
    ws.SetReadLimit(maxMessageSize)
    ws.SetReadDeadline(time.Now().Add(pongWait))
    ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })

    w, err := os.OpenFile(FileI, os.O_WRONLY, 0600)
    if err != nil {
        log.Println("ERROR opening file:", err)
        return
    }

    for {
        _, message, err := ws.ReadMessage()
        if err != nil {
            break
        }
        message = append(message, '\n')
        if _, err := w.Write(message); err != nil {
            break
        }
    }
}

func pumpStdout(ws *websocket.Conn, FileO string, done chan struct{}) {
    // Open file once for all operations
    fFileO, err := os.Open(FileO)
    if err != nil {
        log.Println("ERROR opening file:", err)
        return
    }
    defer fFileO.Close()

    // Create file watcher
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        log.Println("ERROR creating watcher:", err)
        return
    }
    defer watcher.Close()

    // Get initial file size
    stat, err := fFileO.Stat()
    if err != nil {
        log.Println("ERROR getting file stats:", err)
        return
    }
    size := stat.Size()

    // Send initial file content in chunks (memory-efficient)
    buf := make([]byte, 1024)
    for {
        n, err := fFileO.Read(buf)
        if n > 0 {
            ws.SetWriteDeadline(time.Now().Add(writeWait))
            if err := ws.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
                log.Println("ERROR sending initial content:", err)
                return
            }
        }
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Println("ERROR reading initial content:", err)
            return
        }
    }

    // Update current position after initial read
    size, err = fFileO.Seek(0, io.SeekCurrent)
    if err != nil {
        log.Println("ERROR getting file position:", err)
        return
    }

    // Start watching for changes
    if err := watcher.Add(FileO); err != nil {
        log.Println("ERROR adding watch:", err)
        return
    }

    // Main event loop with proper exit handling
    defer close(done)
    defer ws.Close()

    loop:
    for {
        select {
        case event, ok := <-watcher.Events:
            if !ok {
                break loop // Watcher closed
            }

            switch {
            // Handle write events
            case event.Has(fsnotify.Write):
                // Check for file truncation
                currentStat, err := fFileO.Stat()
                if err != nil {
                    log.Println("ERROR checking file stats:", err)
                    break loop
                }

                if currentStat.Size() < size {
                    size = 0 // Reset position if truncated
                }

                // Read new content
                n, err := fFileO.Read(buf)
                if err != nil && err != io.EOF {
                    log.Println("ERROR reading new content:", err)
                    break loop
                }

                if n > 0 {
                    size += int64(n)
                    ws.SetWriteDeadline(time.Now().Add(writeWait))
                    if err := ws.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
                        log.Println("ERROR sending new content:", err)
                        break loop
                    }
                }

            // Handle file rotation/recreation
            case event.Has(fsnotify.Remove), event.Has(fsnotify.Rename):
                // Reopen file after rotation
                fFileO.Close()
                for i := range 10 { // Retry with backoff
                    time.Sleep(time.Duration(i*100) * time.Millisecond)
                    newFile, err := os.Open(FileO)
                    if err == nil {
                        fFileO = newFile
                        size = 0
                        watcher.Add(FileO) // Re-add watch
                        break
                    }
                }
                if fFileO == nil {
                    log.Println("ERROR reopening file after rotation")
                    break loop
                }
            }

        case err, ok := <-watcher.Errors:
            if !ok {
                break loop
            }
            log.Println("Watcher ERROR:", err)
        }
    }

    // Graceful WebSocket closure
    ws.SetWriteDeadline(time.Now().Add(writeWait))
    ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
    time.Sleep(closeGracePeriod)
}

func ping(ws *websocket.Conn, done chan struct{}) {
    ticker := time.NewTicker(pingPeriod)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            if err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
                log.Println("ping:", err)
            }
        case <-done:
            return
        }
    }
}
