package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

type MinecraftServer struct {
    DataPath    string
    javaCmd     string
    args        []string

    FileI       string
    FileO       string
}

func NewMC(dataPath, jarFile string, config McConfig) (*MinecraftServer, error) {
    fifoFile := dataPath + "/fifo"
    _, err := os.Stat(fifoFile)
    if err != nil && os.IsNotExist(err) {
        if err := syscall.Mkfifo(fifoFile, 0640); err != nil {
            return nil, err
        }
    }

    args := strings.Split(config.JavaArgs, " ")
    args = append(args, "-jar", jarFile)
    args = append(args, strings.Split(config.JarOpts, " ")...)

    if IsMCRunningBackground(dataPath) {
        return &MinecraftServer{
            dataPath, config.JavaCmd, args, fifoFile, dataPath + "/logs/latest.log",
        }, nil
    }

    return &MinecraftServer{
        dataPath, config.JavaCmd, args, fifoFile, dataPath + "/logs/latest.log",
    }, nil
}

func (c *MinecraftServer) StartMCForeground() error {
    minecraft := exec.Command(c.javaCmd, c.args...)
    minecraft.Dir = c.DataPath
    minecraft.Stdin = os.Stdin
    minecraft.Stdout = os.Stdout

    procDone, gracDone := make(chan bool, 1), make(chan bool, 1)
    go gracefulShutdown(minecraft, procDone, gracDone)

    err := minecraft.Start(); if err != nil { return err }
    err = minecraft.Wait()  ; if err != nil { return err }

    close(procDone)
    <-gracDone

    return nil
}

func (c *MinecraftServer) StartMCBackground() error {
    tail := exec.Command("tail", "-f", c.FileI)
    minecraft := exec.Command(c.javaCmd, c.args...)
    minecraft.Dir = c.DataPath

    var err error
    minecraft.Stdin, err = tail.StdoutPipe()
    if err != nil { return err }

    sysProcAttr := &syscall.SysProcAttr{Setpgid: true, Pgid: 0}
    tail.SysProcAttr = sysProcAttr
    minecraft.SysProcAttr = sysProcAttr

    if err := tail.Start()
    err != nil { return err }
    if err := minecraft.Start()
    err != nil { return err }

    if err := writePIDsToFile(c.DataPath, tail.Process.Pid, minecraft.Process.Pid)
    err != nil { return err }

    if err := tail.Process.Release()
    err != nil { return err }
    if err := minecraft.Process.Release()
    err != nil { return err }

    return nil
}

func (c *MinecraftServer) StopMC() error {
    tailPID, mcPID, err := readPIDsFromFile(c.DataPath); if err != nil { return err }

    tailProc, err := os.FindProcess(tailPID); if err != nil { return err }
    minecraftProc, err := os.FindProcess(mcPID); if err != nil { return err }

    err = errors.Join(tailProc.Signal(syscall.SIGINT))
    err = errors.Join(minecraftProc.Signal(syscall.SIGINT))
    if err != nil { return err }

    return wait(tailProc, minecraftProc)
}

func gracefulShutdown(cmd *exec.Cmd, procDone, gracDone chan bool) {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()
    select {
    case <-ctx.Done():
        if cmd.Process != nil {
            if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
                log.Printf("Failed to send SIGINT to Minecraft process: %v", err)
            }
        }
    case <-procDone:
    }
    gracDone <- true
}

func RemoveOldWorld(dataPath string) error {
    files, err := filepath.Glob(dataPath + "/world*")
    if err != nil { return err }

    files = append(files, "ops.json")
    files = append(files, "permissions.yml")
    files = append(files, "usercaches.json")

    for _, f := range files {
        if err := os.RemoveAll(f); err != nil {
            return err
        }
    }

    return nil
}

func IsMCRunningBackground(dataPath string) bool {
    tailPID, mcPID, err := readPIDsFromFile(dataPath)
    if err != nil { return false }

    tailProc, err := os.FindProcess(tailPID); if err != nil { return false }
    if err = tailProc.Signal(syscall.Signal(0)); err != nil { return false }
    minecraftProc, err := os.FindProcess(mcPID)  ; if err != nil { return false }
    if err = minecraftProc.Signal(syscall.Signal(0)); err != nil { return false }

    return true
}

func wait(tailProc, minecraftProc *os.Process) error {
    var wg sync.WaitGroup
    var tailErr, minecraftErr error

    if tailProc != nil {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _, tailErr = tailProc.Wait()
        }()
    }

    if minecraftProc != nil {
        _, minecraftErr = minecraftProc.Wait()
    }

    wg.Wait()
    return errors.Join(tailErr, minecraftErr)
}

func writePIDsToFile(dataPath string, tailPID, minecraftPID int) error {
    return os.WriteFile(
        dataPath + "/pid",
        fmt.Appendf(nil, "%d\n%d", tailPID, minecraftPID),
        0640,
    )
}

func readPIDsFromFile(dataPath string) (int, int, error) {
    pidBytes, err := os.ReadFile(dataPath + "/pid")
    if err != nil { return -1, -1, err }

    pidStr := strings.Split(string(pidBytes), "\n")

    tailPID, err := strconv.Atoi(pidStr[0]); if err != nil { return -1, -1, err }
    mcPID  , err := strconv.Atoi(pidStr[1]); if err != nil { return -1, -1, err }

    return tailPID, mcPID, nil
}
