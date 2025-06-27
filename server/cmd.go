package main

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type MCcmd struct {
    *exec.Cmd
}

func NewMCcmd(rootDir, javaCmd, javaArgs, jarPath, jarOpts string) (*MCcmd, error) {
    cmdPath, err := exec.LookPath(javaCmd)
    if err != nil { return nil, err }

    args := strings.Split(javaArgs, " ")
        args = append(args, "-jar", jarPath)
        args = append(args, strings.Split(jarOpts, " ")...)

    cmd := exec.Command(cmdPath, args...)
    cmd.Dir = rootDir

    return &MCcmd{cmd}, nil
}

func (c *MCcmd) Launch() error {
    err := c.Start()
    if err != nil {
        return err
    }
    return nil
}

func (c *MCcmd) StopProcess() {
    c.Process.Signal(syscall.SIGINT)
    c.Wait()
    if c.Process != nil {
        <-time.After(time.Second)
        c.Process.Signal(os.Kill)
    }
}

