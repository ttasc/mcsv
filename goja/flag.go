package main

import (
	"errors"
	"flag"
)

type Flags struct {
    interactive *bool
    socket      *bool
    rootDir     *string
    javaCmd     *string
    javaMem     *int
    jarFile     *string
    jarOpts     *string
}

func SetFlags() Flags {
    f := Flags{
        flag.Bool  ("i", false     , "Keep Minecraft running on console. Can't be used with SOCKET mode"),
        flag.Bool  ("s", false     , "Create a unix socket server. Can't be used with INTERACTIVE mode"),
        flag.String("d", "."       , "Specifies the root directory containing the JAR file and server's data"),
        flag.String("c", "java"    , "Specify java command"),
        flag.Int   ("m", 2048      , "Specifies the amount of memory to allocate for Minecraft in MB"),
        flag.String("j", ""        , "Specify the JAR file name to launch. It must be in the root directory"),
        flag.String("o", "--nogui" , "Specifies additional options to pass to the Minecraft server"),
    }
    flag.Parse()
    return f
}

func (f Flags) Validate() error {
    if *f.jarFile == "" {
        return errors.New("No jar file specified. Please provide a jar file using the -jar flag")
    }

    if *f.javaMem < 1024 {
        return errors.New("Memory must be at least 1024MB")
    }

    if *f.interactive && *f.socket {
        return errors.New("Can't use interactive and socket mode at the same time")
    }

    return nil
}
