package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"syscall"
)

func main() {
    flags := ParseFlags()
    if err := flags.Validate(); err != nil {
        log.Fatal("Parse flags error: ", err)
    }

    dataDir := *flags.dataDir
    jarFile := *flags.jarFile
    config, err := ReadConfig(*flags.configFile)
    if err != nil {
        log.Fatal("Read config error: ", err)
    }

    fifoFile := dataDir + "/fifo"
    logsFile := dataDir + "/logs/latest.log"

    os.Remove(fifoFile)
    err = syscall.Mkfifo(fifoFile, 0640)
    if err != nil {
        log.Fatal("Mkfifo error: ", err)
    }

    mcCmd, err := NewMCcmd(dataDir, jarFile, config.Minecraft, fifoFile)
    if err != nil {
        log.Fatal("NewMCcmd error: ", err)
    }

    switch {
    case *flags.detach:
        mcCmd.StartCmd(true) // true
    case *flags.attach:
        // TODO: cli.go
    case *flags.web:
        web := NewWebServer(config.Webserver, mcCmd, fifoFile, logsFile)
        done := make(chan bool, 1) // Create a done channel to signal when the shutdown is complete
        go web.GracefulShutdown(done) // Run graceful shutdown in a separate goroutine
        fmt.Println("Starting web server on port", config.Webserver.Port)
        err := web.Server.ListenAndServe()
        if err != nil && err != http.ErrServerClosed {
            panic(fmt.Sprintf("http server error: %s", err))
        }
        <-done // Wait for the graceful shutdown to complete
        log.Println("Graceful shutdown complete.")
    default:
        mcCmd.StartCmd(false)
    }
}
