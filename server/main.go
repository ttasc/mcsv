package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
    mutex sync.RWMutex
    waitgroup sync.WaitGroup
)

const (
    SocketFileName = "mcsv.sock"
)

func main() {
    args := SetFlags()

    if err := args.Validate(); err != nil {
        fmt.Printf("Error:\n\t%s\n\n", err)
        flag.Usage()
        return
    }

    // Create a log logFile
    logFile, err := os.OpenFile("gova.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        log.Fatal("Failed to open log file:", err)
        return
    }
    defer logFile.Close()
    logOutput := io.MultiWriter(logFile, os.Stdout)
    // Set output of logs to file
    log.SetOutput(logOutput)

    // Add memory arguments to the java command
    javaArgs := fmt.Sprintf("-Xmx%dM -Xms%dM %s", *args.javaMem, *args.javaMem, "-XX:+AlwaysPreTouch -XX:+DisableExplicitGC -XX:+ParallelRefProcEnabled -XX:+PerfDisableSharedMem -XX:+UnlockExperimentalVMOptions -XX:+UseG1GC -XX:G1HeapRegionSize=8M -XX:G1HeapWastePercent=5 -XX:G1MaxNewSizePercent=40 -XX:G1MixedGCCountTarget=4 -XX:G1MixedGCLiveThresholdPercent=90 -XX:G1NewSizePercent=30 -XX:G1RSetUpdatingPauseTimePercent=5 -XX:G1ReservePercent=20 -XX:InitiatingHeapOccupancyPercent=15 -XX:MaxGCPauseMillis=200 -XX:MaxTenuringThreshold=1 -XX:SurvivorRatio=32 -Dusing.aikars.flags=https://mcflags.emc.gs -Daikars.new.flags=true")

    // Create a command to launch the minecraft server
    cmd, err := NewMCcmd(*args.rootDir, *args.javaCmd, javaArgs, *args.jarFile, *args.jarOpts)
    if err != nil { log.Fatal(err); return }

    // Create the server
    server, err := NewServer(cmd)
    if err != nil { log.Fatal(err); return }
    procDoneCh := make(chan bool, 1) // Create a done channel to signal when the process is done
    waitgroup.Add(1)
    go gracefulShutdown(cmd, server, logFile, procDoneCh)

    switch {
    case *args.interactive:
        cmd.Stdin = os.Stdin
        cmd.Stdout = os.Stdout
    case *args.socket:
        go func() {
            fmt.Printf("Unix-Socket listening on: %s/%s\n\n", server.socketDir, SocketFileName)
            err = server.ListenAndServe()
            if err != nil && err != http.ErrServerClosed {
                panic(fmt.Sprintf("http server error: %s", err))
            }
        }()
    }

    // Countdown to start server
    // countdown()

    // Launch the minecraft server
    err = cmd.Launch()
    if err != nil { log.Fatal(err); return }
    defer cmd.StopProcess()

    waitgroup.Add(1)
    wait(cmd, procDoneCh)

    waitgroup.Wait()

    log.Println("Graceful shutdown complete.")
}

func wait(cmd *MCcmd, done chan bool) {
    defer waitgroup.Done()
    cmd.Wait()
    done <- true
}

func gracefulShutdown(cmd *MCcmd, server *Server, logFile *os.File, done chan bool) {
    defer waitgroup.Done()

    // Create context that listens for the interrupt signal from the OS.
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    select {
    // Listen for the interrupt signal.
    case <-ctx.Done():
        log.Println("\nShutting down...")

        log.Println("Stopping minecraft server")
        if cmd.Process != nil {
            cmd.StopProcess()
        }

        log.Println("Closing unix socket")
        if server != nil {
            server.CloseServer()
        }

        log.Println("Closing log file")
        if logFile != nil {
            logFile.Close()
        }

    case <-done:
        log.Println("Closing unix socket")
        if server != nil {
            server.CloseServer()
        }
        return
    }
}

func countdown() {
    time.Sleep(time.Second)
    print("Starting server in... ")
    for i := 10; i > 0; i-- {
        time.Sleep(time.Second)
        print(i, " ")
    }
    println("\nGo!")
}
