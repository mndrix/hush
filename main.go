package hush // import "github.com/mndrix/hush"

import (
	"fmt"
	"os"
)

// Main implements the main() function of the hush command line tool.
func Main() {
	if os.Args[1] == "init" {
		err := CmdInit(os.Stderr, os.Stdin)
		if err != nil {
			die("%s", err.Error())
		}
		return
	}

	tree, err := LoadTree()
	if os.IsNotExist(err) {
		filename, _ := HushPath()
		fmt.Fprintf(os.Stderr, "hush file does not exist: %s\n", filename)
		fmt.Fprintf(os.Stderr, "Maybe you need to run 'hush init'?\n")
		os.Exit(1)
	}
	if err != nil {
		die("%s", err.Error())
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
		warnings, err = CmdImport(os.Stdin, tree)
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
		v, err := CaptureValue(os.Args[3])
		if err != nil {
			die("%s", err.Error())
		}
		err = CmdSet(os.Stdout, tree, p, v)
	default:
		die("Usage: hush ...")
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
