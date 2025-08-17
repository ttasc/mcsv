package main

import (
	"errors"
	"flag"
	"os"
)

type Flags struct {
    detach      *bool
    attach      *bool
    web         *bool
    dataPath    *string
    jarFile     *string
    configFile  *string
}

func ParseFlags() Flags {
    f := Flags{
        flag.Bool  ("d", false  , "Detach from the terminal and run the process in the background"),
        flag.Bool  ("a", false  , "Attach local standard input, output, and error streams to a process's running"),
        flag.Bool  ("w", false  , "Start a web server for remote control"),
        flag.String("p", ""     , "Specify the Minecraft data path"),
        flag.String("j", ""     , "Specify the jar file"),
        flag.String("c", ""     , "Specify a configuration file"),
    }
    flag.Parse()
    return f
}

func (f Flags) Validate() error {
    if *f.detach && *f.attach {
        return errors.New("cannot specify both -d and -a")
    }
    if *f.detach && *f.web {
        return errors.New("cannot specify both -d and -w")
    }
    if *f.attach && *f.web {
        return errors.New("cannot specify both -a and -w")
    }

    if !*f.attach && *f.jarFile == "" {
        return errors.New("must specify a jar file")
    } else if _, err := os.Stat(*f.jarFile); err != nil {
        return err
    }

    if *f.dataPath == "" {
        return errors.New("must specify a data path")
    } else if _, err := os.Stat(*f.dataPath); err != nil {
        return err
    }

    if *f.configFile == "" {
        *f.configFile = *f.dataPath + "/craftt/config.json"
    } else if _, err := os.Stat(*f.configFile); err != nil {
        return err
    }

    return nil
}
