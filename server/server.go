package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
)

type unixSocket struct {
    listener    net.Listener
    socketDir   string
}

type Clients map[net.Conn]bool

type Server struct {
    unixSocket
    Clients Clients

    mutex sync.RWMutex

    writer io.Writer
    reader io.Reader
}

func NewServer(cmd *MCcmd) (*Server, error) {
    cmdOutputPipe, err := cmd.StdoutPipe(); if err != nil { return nil, err }
    cmdInputPipe , err := cmd.StdinPipe() ; if err != nil { return nil, err }

    socketDir, err := os.MkdirTemp("", SocketFileName)
    if err != nil {
        return nil, err
    }

    return &Server{
        unixSocket: unixSocket{ socketDir: socketDir },
        writer: cmdInputPipe,
        reader: cmdOutputPipe,
    }, nil
}

func (s *Server) ListenAndServe() error {
    sockfile := s.socketDir + "/mcsv.sock"
    ln, err := net.Listen("unix", sockfile)
    s.listener = ln
    s.Clients = make(map[net.Conn]bool)
    if err != nil {
        return err
    }

    return s.Serve()
}

func (s *Server) Serve() error {
    go s.broadcast(s.reader)
    for {
        conn, err := s.listener.Accept()
        if err != nil {
            if strings.Contains(err.Error(), "use of closed network connection") {
                break
            }
            log.Println("Unix-Socket: Accept Failed:", err)
            continue
        }
        s.Clients[conn] = true
        go s.getInputFromClients(conn, s.writer)
    }
    return nil
}

func (s *Server) CloseServer() {
    if s.listener != nil {
        s.listener.Close()
    }
    os.RemoveAll(s.socketDir)
}

func (s *Server) getInputFromClients(conn net.Conn, w io.Writer) {
    defer func() {
        s.mutex.Lock()
        delete(s.Clients, conn)
        conn.Close()
        s.mutex.Unlock()
    }()
    for {
        buf := make([]byte, 4096)
        n, err := conn.Read(buf)
        if err != nil {
            break
        }

        mutex.Lock()

        _, err = w.Write(buf[:n])
        if err != nil {
            log.Println("write to cmd stdin:", err)
            break
        }

        mutex.Unlock()
    }
}

func (s *Server) broadcast(r io.Reader) {
    scanner := bufio.NewScanner(r)
    for scanner.Scan() {
        for conn := range s.Clients {
            if _, err := conn.Write(append(scanner.Bytes(), '\n')); err != nil {
                log.Println("broadcast write:", err)
            }
            /*
                TODO: Multi-Thread still not working
                When one client send incorret command to MC
                Only the client sender itself receives the message in wrong order
            */
            // msg := append(scanner.Bytes(), '\n')
            // go func(conn net.Conn) {
            //     buf := make([]byte, len(msg))
            //     copy(buf, msg)
            //     if _, err := conn.Write(buf); err != nil {
            //         log.Println("broadcast write:", err)
            //     }
            // }(conn)
        }
    }
    if scanner.Err() != nil {
        log.Println("Bradcast: scan:", scanner.Err())
    }
}

