package hush

import (
	"errors"
	"io"
)

// CmdSet sets the value for a given path in tree.
//
// This function implements "hush set"
func CmdSet(w io.Writer, tree *Tree, p Path, v *Value) error {
	if p.IsConfiguration() {
		return errors.New("Can't set a configuration path")
	}
	if p.IsChecksum() {
		return errors.New("Can't set file checksum manually")
	}
	tree.set(p, v)
	t := tree.Filter(p.Parent().String())
	t.Print(w)
	return tree.Save()
}
