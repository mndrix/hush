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
	if os.Args[1] == "init" {
		err := CmdInit(os.Stderr, os.Stdin)
		if err != nil {
			die("%s", err.Error())
		}
		os.Exit(0)
	}

	tree, err := LoadTree()
	if os.IsNotExist(err) {
		filename, _ := hushPath()
		fmt.Fprintf(os.Stderr, "hush file does not exist: %s\n", filename)
		fmt.Fprintf(os.Stderr, "Maybe you need to run 'hush init' first?\n")
		os.Exit(1)
	}
	if err != nil {
		die("%s\n", err.Error())
	}

	// prepare for encryption and decryption
	err = setPassphrase(tree)
	if err != nil {
		die("%s", err.Error())
	}

	// handle most commands
	switch os.Args[1] {
	case "export": // hush export
		err = CmdExport(os.Stdout, tree)
	case "import": // hush import
		var warnings []string
		warnings, err = CmdImport(os.Stdout, os.Stdin, tree)
		for _, warning := range warnings {
			warn(warning)
		}
	case "ls": // hush ls foo/bar
		if len(os.Args) < 3 {
			tree.Print(os.Stdout)
			return
		}
		err = CmdLs(os.Stdout, tree, os.Args[2])
	case "rm": // hush rm paypal.com/personal
		paths := make([]Path, len(os.Args)-2)
		for i := 2; i < len(os.Args); i++ {
			paths[i-2] = NewPath(os.Args[i])
		}
		err = CmdRm(tree, paths)
	case "set": // hush set paypal.com/personal/user john@example.com
		p := NewPath(os.Args[2])
		v, err := captureValue(os.Args[3])
		if err != nil {
			die("%s\n", err.Error())
		}
		err = CmdSet(os.Stdout, tree, p, v)
	default:
		die("Usage: hum ...\n")
	}
	if err != nil {
		die("%s", err.Error())
	}
}

func setPassphrase(t *Tree) error {
	password, err := AskPassword(os.Stderr, "Password")
	if err != nil {
		return err
	}

	return t.SetPassphrase(password)
}

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
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

// hushPath returns the filename of this user's hush file whether it
// exists or not. If the file doesn't exist, it also returns an error
// for which os.IsNotExist() is true.
func hushPath() (string, error) {
	if env := os.Getenv("HUSH_FILE"); env != "" {
		_, err := os.Stat(env)
		return env, err
	}

	home, err := home()
	if err != nil {
		return "", err
	}
	filename := path.Join(home, ".hush")
	f, err := filepath.EvalSymlinks(filename)
	if err == nil {
		filename = f
	}
	return filename, err
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
