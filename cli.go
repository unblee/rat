package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// CLI is the command line object
type CLI struct {
	outStream io.Writer
	errStream io.Writer
	fatalLog  *log.Logger
	cfg       *Config
}

func newCLI(outStream, errStream io.Writer, args []string) (*CLI, error) {
	cfg, err := loadConfig(outStream, errStream, args)
	if err != nil {
		return nil, err
	}
	return &CLI{
		outStream: outStream,
		errStream: errStream,
		fatalLog:  newFatalLogger(errStream),
		cfg:       cfg,
	}, nil
}

func newFatalLogger(errStream io.Writer) *log.Logger {
	return log.New(errStream, "fatal: ", 0)
}

// main process
func (c *CLI) run() int {
	if c.cfg.showVersion {
		return c.outputVersion()
	}

	if c.cfg.showList {
		return c.outputList()
	}

	// copy boilerplate-name to project-name
	err := c.copyDir()
	if err != nil {
		c.fatalLog.Println(err)
		return exitCodeError
	}

	return exitCodeOK
}

func (c *CLI) outputVersion() int {
	fmt.Fprintln(c.outStream, "rat version "+VERSION)
	return exitCodeOK
}

func (c *CLI) outputList() int {
	blist, err := c.cfg.blplList()
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
