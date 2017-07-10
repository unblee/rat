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
	fatalLog  *log.Logger
	cfg       *Config
}

// Config is the command line config
type Config struct {
	showList        bool
	showVersion     bool
	ratRoot         string
	ratSelectCmd    string
	boilerplateName string
	projectPath     string
}

// main process
func (c *CLI) run() int {
	// --version
	// show version
	if c.cfg.showVersion {
		return c.showVersion()
	}

	// --list
	// show boilerplate list
	if c.cfg.showList {
		return c.showList()
	}

	// copy boilerplate-name to project-name
	err := c.copyDir()
	if err != nil {
		c.fatalLog.Println(err)
		return exitCodeError
	}

	return exitCodeOK
}

// set options, environment values and arguments to CLI
func newCLI(outStream, errStream io.Writer, args []string) (*CLI, error) {
	cli := &CLI{
		outStream: outStream,
		errStream: errStream,
		fatalLog:  newFatalLogger(errStream),
	}

	// cli config vars
	var (
		showList        bool
		showVersion     bool
		ratSelectCmd    string
		ratRoot         string
		boilerplateName string
		projectPath     string
	)

	flags := flag.NewFlagSet(NAME, flag.ContinueOnError)
	flags.SetOutput(outStream)

	// parsing flags
	flags.Usage = func() {
		fmt.Fprintln(cli.outStream, helpText)
		os.Exit(exitCodeOK)
	}
	flags.BoolVar(&showList, "list", false, "")
	flags.BoolVar(&showList, "l", false, "")
	flags.BoolVar(&showVersion, "version", false, "")
	flags.BoolVar(&showVersion, "v", false, "")
	flags.Parse(args[1:])

	// set options
	cli.setOption(showList, showVersion)

	// set environment values
	ratSelectCmd = os.Getenv("RAT_SELECT_CMD") // if user do not use a selection filter, this value can be empty
	ratRoot = os.Getenv("RAT_ROOT")
	if ratRoot == "" {
		return nil, fmt.Errorf("Please set 'RAT_ROOT' environment value")
	}
	if err := cli.setEnv(ratRoot, ratSelectCmd); err != nil {
		return nil, err
	}

	// set arguments
	switch flags.NArg() {
	case 0:
		if flags.NFlag() == 0 {
			return nil, fmt.Errorf("Please set 'project-name'\n %s", helpText)
		}
	case 1:
		boilerplateName = ""
		projectPath = flags.Arg(0)
	case 2:
		boilerplateName = flags.Arg(0)
		projectPath = flags.Arg(1)
	default:
		return nil, errors.New("Too many arguments")
	}
	if err := cli.setArg(boilerplateName, projectPath); err != nil {
		return nil, err
	}

	return cli, nil
}

func newFatalLogger(errStream io.Writer) *log.Logger {
	return log.New(errStream, "fatal: ", 0)
}

// set the command line options
func (c *CLI) setOption(showList, showVersion bool) {
	c.cfg.showList = showList
	c.cfg.showVersion = showVersion
}

// set the command line environment values
func (c *CLI) setEnv(ratRoot, ratSelectCmd string) error {
	// expand path
	ratRoot, err := homedir.Expand(ratRoot)
	if err != nil {
		return err
	}
	ratRoot = os.ExpandEnv(ratRoot)

	// delete the suffix directory separator to unify the handling of the path
	ratRoot = strings.TrimSuffix(ratRoot, string(filepath.Separator))

	c.cfg.ratRoot = ratRoot
	c.cfg.ratSelectCmd = ratSelectCmd

	return nil
}

// set the command line arguments
func (c *CLI) setArg(boilerplateName, projectPath string) error {
	// expand path
	projectPath, err := homedir.Expand(projectPath)
	if err != nil {
		return err
	}
	projectPath = os.ExpandEnv(projectPath)

	// select boilerplate
	if boilerplateName == "" {
		boilerplateName, err = c.selectBlpl()
		if err != nil {
			return err
		}
	}

	c.cfg.boilerplateName = boilerplateName
	c.cfg.projectPath = projectPath

	return nil
}

func (c *CLI) showVersion() int {
	fmt.Fprintln(c.outStream, "rat version "+VERSION)
	return exitCodeOK
}

func (c *CLI) showList() int {
	blist, err := c.blplList()
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

// list of boilerplate directries
func (c *CLI) blplList() ([]string, error) {
	// ls ratRoot
	dirs, err := ioutil.ReadDir(c.cfg.ratRoot)
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
func (c *CLI) selectBlpl() (string, error) {
	list, err := c.blplList()
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = c.runSelect(strings.NewReader(strings.Join(list, "\n")), &buf)
	if err != nil {
		return "", err
	}
	if buf.Len() == 0 {
		return "", errors.New("No boilerplate selected")
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

// run selector command
func (c *CLI) runSelect(r io.Reader, w io.Writer) error {
	if c.cfg.ratSelectCmd == "" {
		return errors.New("Please set 'RAT_SELECT_CMD' environment value")
	}
	if !cmdExists(c.cfg.ratSelectCmd) {
		return fmt.Errorf("Not exists '%s' command", c.cfg.ratSelectCmd)
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", c.cfg.ratSelectCmd)
	} else {
		cmd = exec.Command("sh", "-c", c.cfg.ratSelectCmd)
	}
	cmd.Stderr = os.Stderr
	cmd.Stdin = r
	cmd.Stdout = w
	return cmd.Run()
}

func fileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return err == nil
}

func cmdExists(cmdname string) bool {
	_, err := exec.LookPath(cmdname)
	return err == nil
}

func (c *CLI) copyDir() error {
	dst := c.cfg.projectPath
	src := filepath.Join(c.cfg.ratRoot, c.cfg.boilerplateName)
	if !fileExists(src) {
		c.fatalLog.Printf("Not exists directory '%s'", src)
	}

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
