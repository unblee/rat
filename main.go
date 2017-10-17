package main

import (
	"fmt"
	"os"
)

const (
	// NAME is this application name
	NAME = "rat"
)

const (
	exitCodeOK = iota
	exitCodeError
)

func main() {
	cli, err := newCLI(os.Stdout, os.Stderr, os.Args)
	if err != nil {
		fmt.Printf("fatal: %s", err)
		os.Exit(exitCodeError)
	}
	os.Exit(cli.run())
}

const helpText = `
NAME:
    rat - Boilerplate manager

USAGE:
    rat [GLOBAL-OPTIONS] [<boilerplate-name>] <project-name>

GLOBAL-OPTIONS:
    --list, -l     Show boilerplate list
    --version, -v  Show version
    --help, -h     Show this message
`
