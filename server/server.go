package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

type unixSocket struct {
    net.Listener
    socketDir string
}

type Server struct {
    Socket unixSocket
    write *os.File
    read  *os.File
}

func NewUnixSocketServer(write, read *os.File) (*Server, error) {
    socketDir, err := os.MkdirTemp("", "mcsv.sock")
    if err != nil {
        return nil, err
    }

    sockfile := socketDir + "/mcsv.sock"
    socket, err := net.Listen("unix", sockfile)
    if err != nil {
        return nil, err
    }

    fmt.Printf("Listening on: %s\n\n", sockfile)

    return &Server{
        Socket: unixSocket{ Listener: socket, socketDir: socketDir },
        write: write,
        read:  read,
    }, nil
}

func (s *Server) Serve() {
    for {
        conn, err := s.Socket.Accept()
        if err != nil {
            if strings.Contains(err.Error(), "use of closed network connection") {
                break
            }
            log.Println("Accept Failed:", err)
            continue
        }
        go handleConnections(conn, s.write, s.read)
    }
}

func (s *Server) CloseServer() {
    s.Socket.Close()
    os.RemoveAll(s.Socket.socketDir)
}

func handleConnections(conn net.Conn, w io.Writer, r io.Reader) {
    defer conn.Close()
    go readFromClientAndWriteToInput(conn, w)
    readOutputAndWriteToClients(conn, r)
}

func readFromClientAndWriteToInput(conn net.Conn, w io.Writer) {
    message := make([]byte, 4096)
    for {
        mutex.Lock()
        n, err := conn.Read(message)
        if err != nil {
            break
        }
        message = message[:n]
        message = append(message, '\n')
        _, err = w.Write(message)
        if err != nil {
            break
        }
        mutex.Unlock()
    }
}

func readOutputAndWriteToClients(conn net.Conn, r io.Reader) {
    scanner := bufio.NewScanner(r)
    for scanner.Scan() {
        _, err := conn.Write(append(scanner.Bytes(), '\n'))
        if err != nil {
            break
        }
    }
    if scanner.Err() != nil {
        log.Println("scan:", scanner.Err())
    }
}
