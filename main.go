package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var fatalLog = log.New(os.Stderr, "fatal: ", 0)

const version = "0.1.0" // application version

const (
	exitCodeOK = iota
	exitCodeError
)

const usageMsg = `
NAME:
    lat - Boilerplate manager

USAGE:
    lat [global-options] [boilerplate-name] project-name

GLOBAL-OPTIONS:
    --list, -l     Show boilerplate list
    --version, -v  Show version
    --help, -h  Show this message
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
	latRoot      string
	latSelectCmd string
}

func newEnv(latRoot, latSelectCmd string) *env {
	if latRoot == "" {
		fatalLog.Fatalln("Please set 'LAT_ROOT' environment value")
	}

	// expand path
	latRoot = os.ExpandEnv(latRoot)

	// delete the suffix directory separator to unify the handling of the path
	latRoot = strings.TrimSuffix(latRoot, string(filepath.Separator))

	if !fileExists(latRoot) {
		fatalLog.Fatalf("Not exists directory '%s'\n", latRoot)
	}

	if latSelectCmd == "" || !cmdExists(latSelectCmd) {
		fatalLog.Fatalf("Not exists '%s' command\n", latSelectCmd)
	}

	return &env{
		latRoot:      latRoot,
		latSelectCmd: latSelectCmd,
	}
}
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func cmdExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// application args
type args struct {
	boilerplateName string
	projectPath     string
}

func newArgs(boilerplateName, projectName string) *args {
	return &args{
		boilerplateName: boilerplateName,
		projectPath:     projectName,
	}
}

// main process
func run(option *option, env *env, args *args) int {
	// --version
	// show version
	if option.showVersion {
		fmt.Println("lat version " + version)
		return exitCodeOK
	}

	// --list
	// show boilerplate list
	if option.showList {
		blist, err := blplList(env.latRoot)
		if err != nil {
			fatalLog.Println(err)
			return exitCodeError
		}

		// print
		for _, name := range blist {
			fmt.Println(name)
		}
		return exitCodeOK
	}

	// select boilerplate
	var srcBoilerplatePath string
	if args.boilerplateName != "" {
		srcBoilerplatePath = filepath.Join(env.latRoot, args.boilerplateName)
	} else {
		bname, err := selectBlpl(env.latRoot, env.latSelectCmd)
		if err != nil {
			fatalLog.Println(err)
			return exitCodeError
		}
		srcBoilerplatePath = filepath.Join(env.latRoot, bname)
	}
	if !fileExists(srcBoilerplatePath) {
		fatalLog.Printf("Not exists directory '%s'", srcBoilerplatePath)
		return exitCodeError
	}

	// copy boilerplate-name to project-name
	dstProjectPath := args.projectPath
	err := copyDir(dstProjectPath, srcBoilerplatePath)
	if err != nil {
		fatalLog.Println(err)
		return exitCodeError
	}

	return exitCodeOK
}

// list of boilerplate directries
func blplList(latRoot string) ([]string, error) {
	// ls latRoot
	dirs, err := ioutil.ReadDir(latRoot)
	if err != nil {
		return nil, err
	}

	if len(dirs) == 0 {
		return nil, errors.New("Not exists boilerplate directories")
	}

	list := make([]string, len(dirs))
	for i := 0; i < len(dirs); i++ {
		list[i] = dirs[i].Name()
	}
	return list, nil
}

// select boilerplate name
func selectBlpl(latRoot, latSelectCmd string) (string, error) {
	if latSelectCmd == "" {
		return "", errors.New("Please set 'LAT_SELECT_CMD' environment value")
	}
	list, err := blplList(latRoot)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = runSelect(latSelectCmd, strings.NewReader(strings.Join(list, "\n")), &buf)
	if err != nil {
		return "", err
	}
	if buf.Len() == 0 {
		return "", errors.New("No boilerplate selected")
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

// run selector command
func runSelect(selectCmd string, r io.Reader, w io.Writer) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", selectCmd)
	} else {
		cmd = exec.Command("sh", "-c", selectCmd)
	}
	cmd.Stderr = os.Stderr
	cmd.Stdin = r
	cmd.Stdout = w
	return cmd.Run()
}

func copyDir(dst, src string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		// e.g.
		// src  = /home/foo
		// path = /home/foo/bar
		//                 /bar
		path = strings.TrimPrefix(path, src)

		// skip src root dir
		if path == "" {
			return nil
		}

		if info.IsDir() { // make dest dir
			dstDir := filepath.Join(dst, path)
			err := os.MkdirAll(dstDir, info.Mode())
			if err != nil {
				return err
			}
		} else { // copy file
			srcFile, err := os.Open(src)
			if err != nil {
				return err
			}
			defer srcFile.Close()

			dstFile, err := os.Create(filepath.Join(dst, path))
			if err != nil {
				return err
			}
			defer dstFile.Close()

			// TODO: hook to find template

			io.Copy(dstFile, srcFile)
		}

		return nil
	})
}

func main() {
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
	latSelectCmd := os.Getenv("LAT_SELECT_CMD")
	latRoot := os.Getenv("LAT_ROOT")
	env := newEnv(latRoot, latSelectCmd)

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
		fatalLog.Println("Too many arguments")
		os.Exit(exitCodeError)
	}
	args := newArgs(boilerplateName, projectPath)

	os.Exit(run(option, env, args))
}
