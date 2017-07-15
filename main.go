package hush // import "github.com/mndrix/hush"

import (
	"fmt"
	"os"
)

// Main implements the main() function of the hush command line tool.
func Main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	// handle init command before loading tree
	switch os.Args[1] {
	case "help":
		CmdHelp(os.Stdout)
		return
	case "init":
		err := CmdInit(os.Stderr, os.Stdin)
		if err != nil {
			die("%s", err.Error())
		}
		return
	}

	// load tree for all other commands
	tree, err := LoadTree()
	if os.IsNotExist(err) {
		filename, _ := HushPath()
		fmt.Fprintf(os.Stderr, "hush file does not exist: %s\n", filename)
		fmt.Fprintf(os.Stderr, "Maybe you need to run 'hush init'?\n")
		os.Exit(1)
	}
	if err == nil {
		err = setPassphrase(tree)
	}
	if err != nil {
		die("%s", err.Error())
	}

	// dispatch to command
	switch os.Args[1] {
	case "export": // hush export
		err = CmdExport(os.Stdout, tree)
	case "import":
		var warnings []string
		warnings, err = CmdImport(os.Stdin, tree)
		for _, warning := range warnings {
			warn(warning)
		}
	case "ls":
		if len(os.Args) < 3 {
			tree.Print(os.Stdout)
			return
		}
		err = CmdLs(os.Stdout, tree, os.Args[2])
	case "rm":
		paths := make([]Path, len(os.Args)-2)
		for i := 2; i < len(os.Args); i++ {
			paths[i-2] = NewPath(os.Args[i])
		}
		err = CmdRm(tree, paths)
	case "set":
		if len(os.Args) < 4 {
			die("Usage: hush set path value")
		}
		p := NewPath(os.Args[2])
		var v *Value
		v, err = CaptureValue(os.Args[3])
		if err != nil {
			die("%s", err.Error())
		}
		err = CmdSet(os.Stdout, tree, p, v)
	default:
		usage()
	}
	if err != nil {
		die("%s", err.Error())
	}
}

func usage() {
	die("Usage: hush [command [arguments]]")
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
