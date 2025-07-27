package main

import (
	"log"
	"net/http"
)

func main() {
    flags := ParseFlags()
    if err := flags.Validate(); err != nil {
        log.Fatal("Parse flags error: ", err)
    }

    dataPath := *flags.dataPath
    jarFile := *flags.jarFile
    config, err := ReadConfig(*flags.configFile)
    if err != nil {
        log.Fatal("Read config error: ", err)
    }

    mcsv, err := NewMC(dataPath, jarFile, config.Minecraft)
    if err != nil {
        log.Fatal("NewMC error: ", err)
    }

    switch {
    case *flags.detach:
        err := mcsv.StartMCBackground()
        if err != nil {
            log.Fatal("StartMC background error: ", err)
        }
    case *flags.attach:
        // TODO: cli.go
    case *flags.web:
        web := NewWebServer(config.WebServer, mcsv)
        done := make(chan bool, 1) // Create a done channel to signal when the shutdown is complete
        go web.GracefulShutdown(done) // Run graceful shutdown in a separate goroutine
        if config.WebServer.TLS.Enable {
            err = web.Server.ListenAndServeTLS(config.WebServer.TLS.CertFile, config.WebServer.TLS.KeyFile)
        } else {
            err = web.Server.ListenAndServe()
        }
        if err != nil && err != http.ErrServerClosed {
            log.Fatal("http server error: ", err)
        }
        <-done // Wait for the graceful shutdown to complete
        log.Println("Graceful shutdown complete.")
    default:
        err := mcsv.StartMCForeground()
        if err != nil {
            log.Fatal("StartMC foreground error: ", err)
        }
    }
}
