package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

type MinecraftCmd struct {
    tail        *exec.Cmd
    minecraft   *exec.Cmd
}

func NewMCcmd(dataDir, jarFile string, config McConfig, fifoFile string) (*MinecraftCmd, error) {
    if fifoFile == "" {
        return nil, errors.New("no fifo file")
    }
    if _, err := os.Stat(fifoFile); err != nil {
        return nil, err
    }

    tail := exec.Command("tail", "-f", fifoFile)
    args := strings.Split(config.JavaArgs, " ")
    args = append(args, "-jar", jarFile)
    args = append(args, strings.Split(config.JarOpts, " ")...)
    minecraft := exec.Command(config.JavaCmd, args...)
    minecraft.Dir = dataDir

    return &MinecraftCmd{tail, minecraft}, nil
}

func (c *MinecraftCmd) StartCmd(detach bool) error {
    if err := c.pipe(detach); err != nil { return err }

    if !detach { return c.minecraft.Run() }

    if err := c.tail.Start(); err != nil { return err }
    if err := c.minecraft.Start(); err != nil { return err }
    return c.detach()
}

func (c *MinecraftCmd) StopCmd() error {
    var err error
    if c.tail.Process != nil {
        err = errors.Join(c.tail.Process.Kill())
    }
    if c.minecraft.Process != nil {
        err = errors.Join(c.minecraft.Process.Kill())
    }
    return err
}

func (c *MinecraftCmd) IsRunning() bool {
    if c.tail.Process == nil {
        return !c.minecraft.ProcessState.Exited()
    }
    return !c.tail.ProcessState.Exited() && !c.minecraft.ProcessState.Exited()
}

func (c *MinecraftCmd) pipe(detach bool) error {
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

func (c *MinecraftCmd) detach() error {
    if err := c.tail.Process.Release(); err != nil { return err }
    return c.minecraft.Process.Release()
}
