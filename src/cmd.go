package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

type MinecraftServer struct {
    DataPath    string
    FileI       string
    FileO       string

    tail        *exec.Cmd
    minecraft   *exec.Cmd
}

func NewMC(dataPath, jarFile string, config McConfig) (*MinecraftServer, error) {
    fifoFile := dataPath + "/fifo"
    _, err := os.Stat(fifoFile)
    if err != nil && os.IsNotExist(err) {
        if err := syscall.Mkfifo(fifoFile, 0640); err != nil {
            return nil, err
        }
    }

    if IsMCRunningBackground(dataPath) {
        return &MinecraftServer{
            dataPath, fifoFile, dataPath + "/logs/latest.log", nil, nil,
        }, nil
    }

    tail := exec.Command("tail", "-f", fifoFile)
    args := strings.Split(config.JavaArgs, " ")
    args = append(args, "-jar", jarFile)
    args = append(args, strings.Split(config.JarOpts, " ")...)
    minecraft := exec.Command(config.JavaCmd, args...)
    minecraft.Dir = dataPath

    return &MinecraftServer{
        dataPath, fifoFile, dataPath + "/logs/latest.log", tail, minecraft,
    }, nil
}

func (c *MinecraftServer) StartMC(detach bool) error {
    if c.tail == nil || c.minecraft == nil {
        return errors.New("Minecraft is already running")
    }

    if err := c.pipe(detach); err != nil { return err }

    if !detach { return c.minecraft.Run() }

    if err := c.tail.Start()     ; err != nil { return err }
    if err := c.minecraft.Start(); err != nil { return err }

    if err := writePIDsToFile(c.DataPath, c.tail.Process.Pid, c.minecraft.Process.Pid)
    err != nil { return err }

    return c.detach()
}

func (c *MinecraftServer) StopMC() error {
    if c.tail == nil && c.minecraft == nil {
        tailPID, mcPID, err := readPIDsFromFile(c.DataPath); if err != nil { return err }
        if _, err = os.FindProcess(tailPID); err != nil { return err }
        if _, err = os.FindProcess(mcPID)  ; err != nil { return err }
        return nil
    }

    var err error
    if c.tail.Process != nil {
        err = errors.Join(c.tail.Process.Signal(syscall.SIGINT))
    }
    if c.minecraft.Process != nil {
        err = errors.Join(c.minecraft.Process.Signal(syscall.SIGINT))
    }
    return err
}

func (c *MinecraftServer) RemoveOldWorld() error {
    files, err := filepath.Glob(c.DataPath + "/world*")
    if err != nil { return err }

    files = append(files, "ops.json")
    files = append(files, "permissions.yml")
    files = append(files, "usercaches.json")

    for _, f := range files {
        if err := os.Remove(f); err != nil {
            return err
        }
    }

    return nil
}

func IsMCRunningBackground(dataPath string) bool {
    tailPID, mcPID, err := readPIDsFromFile(dataPath)
    if err != nil { return false }

    _, err = os.FindProcess(tailPID); if err != nil { return false }
    _, err = os.FindProcess(mcPID)  ; if err != nil { return false }

    return true
}

func (c *MinecraftServer) pipe(detach bool) error {
    var err error
    if detach {
        c.minecraft.Stdin, err = c.tail.StdoutPipe()
        return err
    } else {
        c.minecraft.Stdin = os.Stdin
        c.minecraft.Stdout = os.Stdout
    }

    return nil
}

func (c *MinecraftServer) detach() error {
    if err := c.tail.Process.Release(); err != nil { return err }
    return c.minecraft.Process.Release()
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
