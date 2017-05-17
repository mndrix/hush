package hush // import "github.com/mndrix/hush"

import (
	"fmt"
	"io/ioutil"
	"os"
	p "path"
	"strings"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Main implements the main() function of the hush command line tool.
func Main() {
	tree, err := LoadTree()
	if err != nil {
		die("%s\n", err.Error())
	}
	//warn("initial tree = %#v\n", tree)

	if len(os.Args) == 1 {
		err = tree.Print()
		if err != nil {
			die("%s\n", err.Error())
		}
		os.Exit(0)
	}

	switch os.Args[1] {
	case "import": // hush import
		warnings, err := CmdImport(tree, os.Stdin)
		if err != nil {
			die("%s\n", err.Error())
		}
		for _, warning := range warnings {
			warn(warning)
		}
	case "ls": // hush ls foo/bar
		if len(os.Args) < 3 {
			tree.Print()
			return
		}
		mainLs(tree, os.Args[2])
	case "set": // hush set paypal.com/personal/user john@example.com
		mainSetValue(tree)
	default:
		die("Usage: hum ...\n")
	}
}

func mainSetValue(tree T) {
	pattern := os.Args[2]
	val, err := captureValue(os.Args[3])
	if err != nil {
		die("%s\n", err.Error())
	}

	p := NewPath(pattern)
	tree.set(p, val)
	tree.Print()
	err = tree.Save()
	if err != nil {
		die("%s\n", err.Error())
	}
}

func mainLs(tree T, pattern string) {
	tree = tree.filter(pattern)
	tree.Print()
}

func isTerminal(file *os.File) bool {
	return terminal.IsTerminal(int(os.Stdin.Fd()))
}

func captureValue(s string) (value, error) {
	if s == "-" {
		if isTerminal(os.Stdout) {
			editor := editor()
			warn("would launch %s to capture value\n", editor)
			return "", nil
		}

		all, err := ioutil.ReadAll(os.Stdin)
		return value(all), err
	}
	return value(s), nil
}

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func warn(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func LoadTree() (T, error) {
	hushPath, err := hushPath()
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(hushPath)
	if os.IsNotExist(err) {
		warn("hush file does not exist. assuming an empty one\n")
		return T{}, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "can't stat hush file")
	}
	// TODO reduce file permissions if they're too loose

	file, err := os.Open(hushPath)
	if err != nil {
		return nil, errors.Wrap(err, "opening hush file")
	}
	hushData, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, errors.Wrap(err, "can't read hush file")
	}

	keys := make(yaml.MapSlice, 0)
	err = yaml.Unmarshal(hushData, &keys)
	if err != nil {
		return nil, errors.Wrap(err, "can't parse hush file")
	}
	tree := newT(keys)
	tree.decrypt()
	return tree, nil
}

func matches(p path, pattern string) bool {
	ps := strings.Split(string(p), "\t")
	patterns := strings.Split(pattern, "/")
	if len(patterns) > len(ps) {
		return false
	}

	for i, pattern := range patterns {
		if !strings.Contains(ps[i], pattern) {
			return false
		}
	}
	return true
}

func home() (string, error) {
	home := os.Getenv("HOME")
	if home == "" {
		return "", errors.New("Point $HOME at your home directory")
	}
	return home, nil
}

func hushPath() (string, error) {
	home, err := home()
	if err != nil {
		return "", err
	}
	return p.Join(home, ".hush"), nil
}

var editorVarNames = []string{
	"HUSH_EDITOR",
	"VISUAL",
	"EDITOR",
}

func editor() string {
	for _, varName := range editorVarNames {
		ed := os.Getenv(varName)
		if ed != "" {
			return ed
		}
	}

	ed := "vi"
	warn("environment configures no editor. defaulting to %s", ed)
	return ed
}
