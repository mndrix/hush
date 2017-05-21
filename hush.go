package hush // import "github.com/mndrix/hush"

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/pkg/errors"
)

// Main implements the main() function of the hush command line tool.
func Main() {
	tree, err := LoadTree()
	if err != nil {
		die("%s\n", err.Error())
	}

	// prepare for encryption and decryption
	err = SetPassphrase(tree)
	if err != nil {
		die("SetPassphrase: %s", err.Error())
	}

	switch os.Args[1] {
	case "export": // hush export
		CmdExport(os.Stdout, tree)
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

func readPassword(w io.Writer, r *os.File, prompt string) ([]byte, error) {
	io.WriteString(w, prompt)
	password, err := terminal.ReadPassword(int(r.Fd()))
	io.WriteString(w, "\n")
	return password, err
}

func SetPassphrase(t *Tree) error {
	password, err := readPassword(os.Stderr, os.Stdin, "Password: ")
	if err != nil {
		return err
	}

	t.SetPassphrase(password)
	return nil
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
