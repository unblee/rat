package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mitchellh/go-homedir"
)

// Config is the command line config
type Config struct {
	showList        bool
	showVersion     bool
	ratRoot         string
	ratSelectCmd    string
	boilerplateName string
	projectPath     string
}

// set options, environment values and arguments to Config
func loadConfig(stdout, errStream io.Writer, args []string) (*Config, error) {
	cfg := new(Config)

	flags := flag.NewFlagSet(NAME, flag.ContinueOnError)
	flags.SetOutput(stdout)

	// set help text
	flags.Usage = func() {
		fmt.Fprintln(stdout, helpText)
		os.Exit(exitCodeOK)
	}

	// set the command line options
	flags.BoolVar(&cfg.showList, "list", false, "")
	flags.BoolVar(&cfg.showList, "l", false, "")
	flags.BoolVar(&cfg.showVersion, "version", false, "")
	flags.BoolVar(&cfg.showVersion, "v", false, "")
	flags.Parse(args[1:])

	// set environment values
	cfg.ratRoot = os.Getenv("RAT_ROOT") // if user do not use a selection filter, this value can be empty
	cfg.ratSelectCmd = os.Getenv("RAT_SELECT_CMD")

	// set arguments
	switch flags.NArg() {
	case 0:
		if flags.NFlag() == 0 {
			return nil, fmt.Errorf("Please set 'project-name'\n %s", helpText)
		}
	case 1:
		cfg.boilerplateName = ""
		cfg.projectPath = flags.Arg(0)
	case 2:
		cfg.boilerplateName = flags.Arg(0)
		cfg.projectPath = flags.Arg(1)
	default:
		return nil, errors.New("Too many arguments")
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// config validation
func (cfg *Config) validate() error {
	// -- ratRoot validation
	// expand path
	ratRoot, err := homedir.Expand(cfg.ratRoot)
	if err != nil {
		return err
	}
	ratRoot = os.ExpandEnv(ratRoot)
	// delete the suffix directory separator to unify the handling of the path
	cfg.ratRoot = strings.TrimSuffix(ratRoot, string(filepath.Separator))

	// -- boilerplateName validation
	if cfg.hasExecSelectCmd() {
		// -- ratSelectCmd validation
		if cfg.ratSelectCmd == "" {
			return errors.New("Please set 'RAT_SELECT_CMD' environment value")
		}
		if !cmdExists(cfg.ratSelectCmd) {
			return fmt.Errorf("Not exists '%s' command", cfg.ratSelectCmd)
		}

		boilerplateName, err := cfg.selectBlpl()
		if err != nil {
			return err
		}
		cfg.boilerplateName = boilerplateName
	}

	// -- projectPath validation
	projectPath, err := homedir.Expand(cfg.projectPath)
	if err != nil {
		return err
	}
	cfg.projectPath = os.ExpandEnv(projectPath)

	return nil
}

// returns true if options and boilerplate name are not specified.
// that is, select command is executed.
func (cfg *Config) hasExecSelectCmd() bool {
	return !(cfg.showList || cfg.showVersion) && cfg.boilerplateName == ""
}

// list of boilerplate directries
func (cfg *Config) blplList() ([]string, error) {
	// ls ratRoot
	dirs, err := ioutil.ReadDir(cfg.ratRoot)
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
func (cfg *Config) selectBlpl() (string, error) {
	list, err := cfg.blplList()
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = cfg.runSelect(strings.NewReader(strings.Join(list, "\n")), &buf)
	if err != nil {
		return "", err
	}
	if buf.Len() == 0 {
		return "", errors.New("No boilerplate selected")
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

// run selector command
func (cfg *Config) runSelect(r io.Reader, w io.Writer) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", cfg.ratSelectCmd)
	} else {
		cmd = exec.Command("sh", "-c", cfg.ratSelectCmd)
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
