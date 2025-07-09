package main

import (
	"errors"
	"flag"
)

type Flags struct {
    interactive *bool
    detach      *bool
    attach      *bool
    web         *bool
}

func SetFlags() Flags {
    f := Flags{
        flag.Bool  ("i", false     , "Keep Minecraft running on console"),
        flag.Bool  ("d", false     , "Detach from the terminal and run the process in the background"),
        flag.Bool  ("a", false     , "Attach local standard input, output, and error streams to a process's running"),
        flag.Bool  ("w", false     , "Start a web server for remote control"),
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

    return nil
}
