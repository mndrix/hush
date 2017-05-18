package hush

import (
	"io"
	"io/ioutil"
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

func CmdSet(w io.Writer, tree T, p Path, v Value) error {
	tree.set(p, v)
	t := tree.filter(p.Parent().AsPattern())
	t.Print(w)
	return tree.Save()
}

func isTerminal(file *os.File) bool {
	return terminal.IsTerminal(int(os.Stdin.Fd()))
}

func captureValue(s string) (Value, error) {
	if s == "-" {
		if isTerminal(os.Stdout) {
			editor := editor()
			warn("would launch %s to capture value\n", editor)
			return "", nil
		}

		all, err := ioutil.ReadAll(os.Stdin)
		return NewValue(string(all)), err
	}
	return NewValue(s), nil
}
