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

	"github.com/mitchellh/go-homedir"
)

// CLI is the command line object
type CLI struct {
	outStream io.Writer
	errStream io.Writer
	option    *Option
	env       *Env
	arg       *Arg
	fatalLog  *log.Logger
}

func newCLI(outStream, errStream io.Writer) *CLI {
	fatalLogger := newFatalLogger(errStream)
	return &CLI{
		outStream: outStream,
		errStream: errStream,
		fatalLog:  fatalLogger,
	}
}

func newFatalLogger(errStream io.Writer) *log.Logger {
	return log.New(errStream, "fatal: ", 0)
}

// Option is the command line options
type Option struct {
	showList    bool
	showVersion bool
}

func (c *CLI) setOption(showList, showVersion bool) {
	c.option = &Option{
		showList:    showList,
		showVersion: showVersion,
	}
}

// Env is the command line environment values
type Env struct {
	ratRoot      string
	ratSelectCmd string
}

func (c *CLI) setEnv(ratRoot, ratSelectCmd string) error {
	if ratRoot == "" {
		return fmt.Errorf("Please set 'RAT_ROOT' environment value")
	}

	// expand path
	ratRoot, err := homedir.Expand(ratRoot)
	if err != nil {
		return err
	}
	ratRoot = os.ExpandEnv(ratRoot)

	// delete the suffix directory separator to unify the handling of the path
	ratRoot = strings.TrimSuffix(ratRoot, string(filepath.Separator))

	if !fileExists(ratRoot) {
		return fmt.Errorf("Not exists directory '%s'", ratRoot)
	}

	if ratSelectCmd == "" || !cmdExists(ratSelectCmd) {
		return fmt.Errorf("Not exists '%s' command", ratSelectCmd)
	}

	c.env = &Env{
		ratRoot:      ratRoot,
		ratSelectCmd: ratSelectCmd,
	}

	return nil
}

// Arg is the command line arguments
type Arg struct {
	boilerplateName string
	projectPath     string
}

func (c *CLI) setArg(boilerplateName, projectPath string) error {
	// expand path
	projectPath, err := homedir.Expand(projectPath)
	if err != nil {
		return err
	}
	projectPath = os.ExpandEnv(projectPath)

	c.arg = &Arg{
		boilerplateName: boilerplateName,
		projectPath:     projectPath,
	}

	return nil
}

// main process
func (c *CLI) run(args []string) int {
	if err := c.init(args); err != nil {
		c.fatalLog.Println(err)
		return exitCodeError
	}

	// --version
	// show version
	if c.option.showVersion {
		return c.showVersion()
	}

	// --list
	// show boilerplate list
	if c.option.showList {
		return c.showList()
	}

	// select boilerplate
	var srcBoilerplatePath string
	if c.arg.boilerplateName != "" {
		srcBoilerplatePath = filepath.Join(c.env.ratRoot, c.arg.boilerplateName)
	} else {
		bname, err := selectBlpl(c.env.ratRoot, c.env.ratSelectCmd)
		if err != nil {
			c.fatalLog.Println(err)
			return exitCodeError
		}
		srcBoilerplatePath = filepath.Join(c.env.ratRoot, bname)
	}
	if !fileExists(srcBoilerplatePath) {
		c.fatalLog.Printf("Not exists directory '%s'", srcBoilerplatePath)
		return exitCodeError
	}

	// copy boilerplate-name to project-name
	dstProjectPath := c.arg.projectPath
	err := copyDir(dstProjectPath, srcBoilerplatePath)
	if err != nil {
		c.fatalLog.Println(err)
		return exitCodeError
	}

	return exitCodeOK
}

// set options, environment values and arguments to cli
func (c *CLI) init(args []string) error {
	flags := flag.NewFlagSet(NAME, flag.ContinueOnError)
	flags.SetOutput(c.outStream)

	// parsing flags
	flags.Usage = func() {
		fmt.Fprintln(c.outStream, helpText)
		os.Exit(exitCodeOK)
	}
	var (
		showList    bool
		showVersion bool
	)
	flags.BoolVar(&showList, "list", false, "")
	flags.BoolVar(&showList, "l", false, "")
	flags.BoolVar(&showVersion, "version", false, "")
	flags.BoolVar(&showVersion, "v", false, "")
	flags.Parse(args[1:])

	// set options
	c.setOption(showList, showVersion)

	// set environment values
	ratSelectCmd := os.Getenv("RAT_SELECT_CMD")
	ratRoot := os.Getenv("RAT_ROOT")
	if err := c.setEnv(ratRoot, ratSelectCmd); err != nil {
		return err
	}

	// set arguments
	var (
		boilerplateName string
		projectPath     string
	)
	switch flags.NArg() {
	case 0:
		if flags.NFlag() == 0 {
			return fmt.Errorf("Please set 'project-name'\n %s", helpText)
		}
	case 1:
		projectPath = flags.Arg(0)
	case 2:
		boilerplateName = flags.Arg(0)
		projectPath = flags.Arg(1)
	default:
		return errors.New("Too many arguments")
	}
	if err := c.setArg(boilerplateName, projectPath); err != nil {
		return err
	}

	return nil
}

func (c *CLI) showVersion() int {
	fmt.Fprintln(c.outStream, "rat version "+VERSION)
	return exitCodeOK
}

func (c *CLI) showList() int {
	blist, err := blplList(c.env.ratRoot)
	if err != nil {
		c.fatalLog.Println(err)
		return exitCodeError
	}

	// print
	for _, name := range blist {
		fmt.Fprintln(c.outStream, name)
	}
	return exitCodeOK
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func cmdExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// list of boilerplate directries
func blplList(ratRoot string) ([]string, error) {
	// ls ratRoot
	dirs, err := ioutil.ReadDir(ratRoot)
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
func selectBlpl(rootPath, selectCmd string) (string, error) {
	if selectCmd == "" {
		return "", errors.New("Please set 'RAT_SELECT_CMD' environment value")
	}
	list, err := blplList(rootPath)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = runSelect(selectCmd, strings.NewReader(strings.Join(list, "\n")), &buf)
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
	os.Mkdir(dst, 0755)
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
			err := os.Mkdir(dstDir, info.Mode())
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
