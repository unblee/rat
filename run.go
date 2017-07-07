package main

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// main process
func run(option *option, env *env, args *args, outStream, errStream io.Writer) int {
	fatalLog := newFatalLogger(errStream)
	stdLog := newStdLogger(outStream)

	// --version
	// show version
	if option.showVersion {
		stdLog.Println("rat version " + version)
		return exitCodeOK
	}

	// --list
	// show boilerplate list
	if option.showList {
		blist, err := blplList(env.ratRoot)
		if err != nil {
			fatalLog.Println(err)
			return exitCodeError
		}

		// print
		for _, name := range blist {
			stdLog.Println(name)
		}
		return exitCodeOK
	}

	// select boilerplate
	var srcBoilerplatePath string
	if args.boilerplateName != "" {
		srcBoilerplatePath = filepath.Join(env.ratRoot, args.boilerplateName)
	} else {
		bname, err := selectBlpl(env.ratRoot, env.ratSelectCmd)
		if err != nil {
			fatalLog.Println(err)
			return exitCodeError
		}
		srcBoilerplatePath = filepath.Join(env.ratRoot, bname)
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
func selectBlpl(ratRoot, ratSelectCmd string) (string, error) {
	if ratSelectCmd == "" {
		return "", errors.New("Please set 'RAT_SELECT_CMD' environment value")
	}
	list, err := blplList(ratRoot)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = runSelect(ratSelectCmd, strings.NewReader(strings.Join(list, "\n")), &buf)
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
