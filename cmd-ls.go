package hush

import "io"

func CmdLs(w io.Writer, tree *Tree, pattern string) error {
	tree = tree.filter(pattern)
	err := tree.Print(w)
	return err
}
