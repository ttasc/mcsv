package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type Process struct {
    *os.Process
}

type MCcmd struct {
    rootDir     string
    javaCmd     string
    args        []string

    stdin   *os.File // Write input to this. It will be sent to java process
    inr     *os.File // For java process to read input sent by stdin
    stdout  *os.File // Read output from this. Java process will write it's output to here
    outw    *os.File // For java process to write output to stdout
}

func NewMCcmd(rootDir, javaCmd, javaArgs, jarPath, jarOpts string) (*MCcmd, error) {
    outr, outw, err := os.Pipe()
    if err != nil {
        log.Fatal(err)
        return nil, err
    }

    inr, inw, err := os.Pipe()
    if err != nil {
        log.Fatal(err)
        return nil, err
    }

    cmdPath, err := exec.LookPath(javaCmd)
    if err != nil {
        return nil, err
    }

    args := strings.Split(javaArgs, " ")
        args = append(args, "-jar", jarPath)
        args = append(args, strings.Split(jarOpts, " ")...)

    return &MCcmd{
        rootDir:    fmt.Sprintf("%s/", rootDir),
        javaCmd:    cmdPath,
        args:       args,

        stdin:      inw,
        inr:        inr,
        stdout:     outr,
        outw:       outw,
    }, nil
}

func (c *MCcmd) Launch() (*Process, error) {
    proc, err := os.StartProcess(c.javaCmd, c.args, &os.ProcAttr{
        Dir:   c.rootDir,
        Files: []*os.File{c.inr, c.outw, c.outw},
    })
    if err != nil {
        return nil, err
    }

    c.inr.Close()
    c.outw.Close()
    return &Process{proc}, nil
}

func (c *MCcmd) CloseIO() {
    c.stdin.Close()
    c.stdout.Close()
}

func (p *Process) StopProcess() {
    p.Signal(syscall.SIGINT)
    if p.Process != nil {
        <-time.After(time.Second)
        p.Signal(os.Kill)
    }
}

