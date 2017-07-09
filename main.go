package main

import (
	"os"
)

const (
	// NAME is this application name
	NAME = "rat"
	// VERSION is this application version
	VERSION = "0.1.0"
)

const (
	exitCodeOK = iota
	exitCodeError
)

func main() {
	cli := newCli(os.Stdout, os.Stderr)
	os.Exit(cli.run(os.Args))
}

const helpText = `
NAME:
    rat - Boilerplate manager

USAGE:
    rat [global-options] [boilerplate-name] project-name

GLOBAL-OPTIONS:
    --list, -l     Show boilerplate list
    --version, -v  Show version
    --help, -h     Show this message
`
