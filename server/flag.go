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
        flag.Bool  ("i"   , false     , "Keep Minecraft Server running on console. Can't be used with SOCKET mode"),
        flag.Bool  ("s"   , false     , "Create a unix socket. Can't be used with INTERACTIVE mode"),
        flag.String("dir" , "."       , "Specifies the root directory containing the server's data and server .jar file"),
        flag.String("java", "java"    , "Specify java command"),
        flag.Int   ("mem" , 2048      , "Specifies the amount of memory to allocate to the minecraft server in MB"),
        flag.String("jar" , ""        , "Specify the jar file name to launch. It must be in the root directory"),
        flag.String("opts", "--nogui" , "Specifies additional options to pass to the minecraft server"),
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
