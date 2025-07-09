package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

type Clients map[net.Conn]bool

type Socket struct {
    listener    net.Listener
    clients     Clients

    mutex       sync.RWMutex

    writer      io.Writer
    reader      io.Reader
}

func NewSocket(cmd *MCcmd) (*Socket, error) {
    cmdOutputPipe, err := cmd.StdoutPipe(); if err != nil { return nil, err }
    cmdInputPipe , err := cmd.StdinPipe() ; if err != nil { return nil, err }

    return &Socket{
        writer: cmdInputPipe,
        reader: cmdOutputPipe,
    }, nil
}

func (s *Socket) ListenAndServe(socketfile string) error {
    ln, err := net.Listen("unix", socketfile)
    s.listener = ln
    s.clients = make(map[net.Conn]bool)
    if err != nil {
        return err
    }

    return s.Serve()
}

func (s *Socket) Serve() error {
    go s.broadcast(s.reader)
    for {
        conn, err := s.listener.Accept()
        if err != nil {
            if strings.Contains(err.Error(), "use of closed network connection") {
                break
            }
            log.Println("Unix-Socket accept failed:", err)
            continue
        }
        s.clients[conn] = true
        go s.getInputFromClients(conn, s.writer)
    }
    return nil
}

func (s *Socket) CloseSocket() {
    if s.listener != nil {
        s.listener.Close()
    }
}

func (s *Socket) getInputFromClients(conn net.Conn, w io.Writer) {
    defer func() {
        s.mutex.Lock()
        delete(s.clients, conn)
        conn.Close()
        s.mutex.Unlock()
    }()
    for {
        buf := make([]byte, 4096)
        n, err := conn.Read(buf)
        if err != nil {
            break
        }
        _, err = w.Write(buf[:n])
        if err != nil {
            log.Println("Failed to write to cmd stdin:", err)
            break
        }
    }
}

func (s *Socket) broadcast(r io.Reader) {
    scanner := bufio.NewScanner(r)
    for scanner.Scan() {
        for conn := range s.clients {
            if _, err := conn.Write(append(scanner.Bytes(), '\n')); err != nil {
                log.Println("Broadcast Failed:", err)
            }
            /*
                TODO: Multi-Thread still not work
                When one client send incorret command to MC
                Only the client sender itself receives the message in wrong order
            */
            // msg := append(scanner.Bytes(), '\n')
            // go func(conn net.Conn) {
            //     buf := make([]byte, len(msg))
            //     copy(buf, msg)
            //     if _, err := conn.Write(buf); err != nil {
            //         log.Println("Broadcast Failed:", err)
            //     }
            // }(conn)
        }
    }
    if scanner.Err() != nil {
        log.Println("Broadcast Failed (scan):", scanner.Err())
    }
}

