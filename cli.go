package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// main process
func (c *cli) run() int {
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
	if c.args.boilerplateName != "" {
		srcBoilerplatePath = filepath.Join(c.env.ratRoot, c.args.boilerplateName)
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
	dstProjectPath := c.args.projectPath
	err := copyDir(dstProjectPath, srcBoilerplatePath)
	if err != nil {
		c.fatalLog.Println(err)
		return exitCodeError
	}

	return exitCodeOK
}

func (c *cli) showVersion() int {
	fmt.Fprintln(c.outStream, "rat version "+VERSION)
	return exitCodeOK
}

func (c *cli) showList() int {
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
