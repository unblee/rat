package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
)

const version = "0.1.0" // application version

const (
	exitCodeOK = iota
	exitCodeError
)

const usageMsg = `
NAME:
    rat - Boilerplate manager

USAGE:
    rat [global-options] [boilerplate-name] project-name

GLOBAL-OPTIONS:
    --list, -l     Show boilerplate list
    --version, -v  Show version
    --help, -h     Show this message
`

// application option
type option struct {
	showList    bool
	showVersion bool
}

func newOption(showList, showVersion bool) *option {
	return &option{
		showList:    showList,
		showVersion: showVersion,
	}
}

// application env
type env struct {
	ratRoot      string
	ratSelectCmd string
}

func newEnv(ratRoot, ratSelectCmd string) *env {
	fatalLog := newFatalLogger(os.Stderr)

	if ratRoot == "" {
		fatalLog.Fatalln("Please set 'RAT_ROOT' environment value")
	}

	// expand path
	ratRoot, err := homedir.Expand(ratRoot)
	if err != nil {
		fatalLog.Fatal(err)
	}
	ratRoot = os.ExpandEnv(ratRoot)

	// delete the suffix directory separator to unify the handling of the path
	ratRoot = strings.TrimSuffix(ratRoot, string(filepath.Separator))

	if !fileExists(ratRoot) {
		fatalLog.Fatalf("Not exists directory '%s'\n", ratRoot)
	}

	if ratSelectCmd == "" || !cmdExists(ratSelectCmd) {
		fatalLog.Fatalf("Not exists '%s' command\n", ratSelectCmd)
	}

	return &env{
		ratRoot:      ratRoot,
		ratSelectCmd: ratSelectCmd,
	}
}

// application args
type args struct {
	boilerplateName string
	projectPath     string
}

func newArgs(boilerplateName, projectPath string) *args {
	fatalLog := newFatalLogger(os.Stderr)

	// expand path
	projectPath, err := homedir.Expand(projectPath)
	if err != nil {
		fatalLog.Fatalln(err)
	}
	projectPath = os.ExpandEnv(projectPath)

	return &args{
		boilerplateName: boilerplateName,
		projectPath:     projectPath,
	}
}

func newFatalLogger(errStream io.Writer) *log.Logger {
	return log.New(errStream, "fatal: ", 0)
}

func main() {
	fatalLog := newFatalLogger(os.Stderr)

	// flag parse
	flag.Usage = func() {
		fmt.Println(usageMsg)
		os.Exit(exitCodeOK)
	}
	showListL := flag.Bool("list", false, "Show boilerplate list")
	showListS := flag.Bool("l", false, "Show boilerplate list")
	showVersionL := flag.Bool("version", false, "Show version")
	showVersionS := flag.Bool("v", false, "Show version")
	flag.Parse()

	// get option
	showList := *showListL || *showListS
	showVersion := *showVersionL || *showVersionS
	option := newOption(showList, showVersion)

	// get env
	ratSelectCmd := os.Getenv("RAT_SELECT_CMD")
	ratRoot := os.Getenv("RAT_ROOT")
	env := newEnv(ratRoot, ratSelectCmd)

	// get args
	var boilerplateName string
	var projectPath string
	switch flag.NArg() {
	case 0:
		if flag.NFlag() == 0 {
			fmt.Println(usageMsg)
			os.Exit(exitCodeError)
		}
	case 1:
		projectPath = flag.Arg(0)
	case 2:
		boilerplateName = flag.Arg(0)
		projectPath = flag.Arg(1)
	default:
		fatalLog.Fatalln("Too many arguments")
	}
	args := newArgs(boilerplateName, projectPath)

	os.Exit(run(option, env, args, os.Stdout, os.Stderr))
}
