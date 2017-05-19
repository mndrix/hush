package hush // import "github.com/mndrix/hush"

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
)

// Main implements the main() function of the hush command line tool.
func Main() {
	tree, err := LoadTree()
	if err != nil {
		die("%s\n", err.Error())
	}
	//warn("initial tree = %#v\n", tree)

	if len(os.Args) == 1 {
		err = tree.Print(os.Stdout)
		if err != nil {
			die("%s\n", err.Error())
		}
		os.Exit(0)
	}

	switch os.Args[1] {
	case "import": // hush import
		warnings, err := CmdImport(os.Stdout, os.Stdin, tree)
		if err != nil {
			die("%s\n", err.Error())
		}
		for _, warning := range warnings {
			warn(warning)
		}
	case "ls": // hush ls foo/bar
		if len(os.Args) < 3 {
			tree.Print(os.Stdout)
			return
		}
		CmdLs(os.Stdout, tree, os.Args[2])
	case "set": // hush set paypal.com/personal/user john@example.com
		p := NewPath(os.Args[2])
		v, err := captureValue(os.Args[3])
		if err != nil {
			die("%s\n", err.Error())
		}
		err = CmdSet(os.Stdout, tree, p, v)
		if err != nil {
			die("%s\n", err.Error())
		}
	default:
		die("Usage: hum ...\n")
	}
}

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func warn(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
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
	filename := path.Join(home, ".hush")
	return filepath.EvalSymlinks(filename)
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
