package hush

import (
	"errors"
	"io"
	"io/ioutil"
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

func CmdSet(w io.Writer, tree *Tree, p Path, v *Value) error {
	if p.IsConfiguration() {
		return errors.New("Can't set a configuration path")
	}
	tree.set(p, v)
	t := tree.filter(p.Parent().String())
	t.Print(w)
	return tree.Save()
}

func isTerminal(file *os.File) bool {
	return terminal.IsTerminal(int(os.Stdin.Fd()))
}

func captureValue(s string) (*Value, error) {
	if s == "-" {
		if isTerminal(os.Stdout) {
			editor := editor()
			warn("would launch %s to capture value\n", editor)
			return nil, nil
		}

		all, err := ioutil.ReadAll(os.Stdin)
		return NewPlaintext(all, Private), err
	}
	return NewPlaintext([]byte(s), Private), nil
}
