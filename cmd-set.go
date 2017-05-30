package hush

import (
	"errors"
	"io"
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
