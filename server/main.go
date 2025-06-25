package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
    mutex sync.RWMutex
    waitgroup sync.WaitGroup
)

func main() {
    var (
        interactive = flag.Bool  ("i"   , false     , "Keep STDIN open")
        verbose     = flag.Bool  ("v"   , false     , "Enable verbose logging to STDIN")
        socket      = flag.Bool  ("s"   , false     , "Run the server using a unix socket")
        rootDir     = flag.String("dir" , "."       , "Specifies the root directory containing the server's data and server .jar file")
        javaCmd     = flag.String("java", "java"    , "Specify java command")
        javaMem     = flag.Int   ("mem" , 2048      , "Specifies the amount of memory to allocate to the minecraft server in MB")
        jarFile     = flag.String("jar" , ""        , "Specify the jar file name to launch. It must be in the root directory")
        jarOpts     = flag.String("opts", "--nogui" , "Specifies additional options to pass to the minecraft server")
    )

    flag.Parse()

    if *jarFile == "" {
        fmt.Println("No jar file specified. Please provide a jar file using the -jar flag")
        return
    }

    defaultArgs := "-XX:+AlwaysPreTouch -XX:+DisableExplicitGC -XX:+ParallelRefProcEnabled -XX:+PerfDisableSharedMem -XX:+UnlockExperimentalVMOptions -XX:+UseG1GC -XX:G1HeapRegionSize=8M -XX:G1HeapWastePercent=5 -XX:G1MaxNewSizePercent=40 -XX:G1MixedGCCountTarget=4 -XX:G1MixedGCLiveThresholdPercent=90 -XX:G1NewSizePercent=30 -XX:G1RSetUpdatingPauseTimePercent=5 -XX:G1ReservePercent=20 -XX:InitiatingHeapOccupancyPercent=15 -XX:MaxGCPauseMillis=200 -XX:MaxTenuringThreshold=1 -XX:SurvivorRatio=32 -Dusing.aikars.flags=https://mcflags.emc.gs -Daikars.new.flags=true"
    javaArgs := defaultArgs
    if *javaMem > 0 {
        javaArgs = fmt.Sprintf("-Xmx%dM -Xms%dM %s", *javaMem, *javaMem, defaultArgs)
    }

    // Create a command to launch the minecraft server
    cmd, err := NewMCcmd(*rootDir, *javaCmd, javaArgs, *jarFile, *jarOpts)
    if err != nil { log.Fatal(err); return }
    defer cmd.CloseIO()

    // Handle Standard I/O if run as a daemon
    if *interactive {
        go io.Copy(cmd.stdin, os.Stdin)
    }
    if *verbose {
        go io.Copy(os.Stdout, cmd.stdout)
    }

    var server *Server
    if *socket {
        // Creat a unix socket server
        server, err = NewUnixSocketServer(cmd.stdin, cmd.stdout)
        if err != nil { log.Fatal(err); return }
        defer server.CloseServer()
        // Handle Socket I/O
        go server.Serve()
    }

    // Launch the minecraft server
    proc, err := cmd.Launch()
    if err != nil { log.Fatal(err); return }
    defer proc.StopProcess()

    // Create a procDoneCh channel to signal when the shutdown is complete
    procDoneCh := make(chan bool, 1)

    waitgroup.Add(1)
    go gracefulShutdown(cmd, proc, server, procDoneCh)

    waitgroup.Add(1)
    wait(proc, procDoneCh)

    waitgroup.Wait()

    log.Println("Graceful shutdown complete.")
}

func wait(proc *Process, done chan bool) {
    defer waitgroup.Done()
    proc.Wait()
    done <- true
}

func gracefulShutdown(cmd *MCcmd, proc *Process, server *Server, done chan bool) {
    defer waitgroup.Done()

    // Create context that listens for the interrupt signal from the OS.
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    select {
    // Listen for the interrupt signal.
    case <-ctx.Done():
        log.Println("Stopping minecraft server")
        proc.StopProcess()
        proc.Wait()

        log.Println("Closing IO file")
        cmd.CloseIO()

        log.Println("Closing unix socket server")
        if server != nil {
            server.CloseServer()
        }
    case <-done:
        return
    }
}
