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

const VERSION = "0.1.0" // application version

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

type cli struct {
	outStream io.Writer
	errStream io.Writer
	option    *option
	env       *env
	args      *args
	fatalLog  *log.Logger
}

func newCLI(outStream, errStream io.Writer, option *option, env *env, args *args) *cli {
	return &cli{
		outStream: outStream,
		errStream: errStream,
		option:    option,
		env:       env,
		args:      args,
		fatalLog:  newFatalLogger(errStream),
	}
}

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
	var (
		showList    bool
		showVersion bool
	)
	flag.BoolVar(&showList, "list", false, "")
	flag.BoolVar(&showList, "l", false, "")
	flag.BoolVar(&showVersion, "version", false, "")
	flag.BoolVar(&showVersion, "v", false, "")
	flag.Parse()

	// get option
	option := newOption(showList, showVersion)

	// get env
	ratSelectCmd := os.Getenv("RAT_SELECT_CMD")
	ratRoot := os.Getenv("RAT_ROOT")
	env := newEnv(ratRoot, ratSelectCmd)

	// get args
	var (
		boilerplateName string
		projectPath     string
	)
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

	cli := newCLI(os.Stdout, os.Stderr, option, env, args)

	os.Exit(cli.run())
}
