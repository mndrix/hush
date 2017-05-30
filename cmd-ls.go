package hush

import "io"

// CmdLs prints to w that portion of tree which matches pattern.
//
// This function implements "hush ls"
func CmdLs(w io.Writer, tree *Tree, pattern string) error {
	tree = tree.Filter(pattern)
	err := tree.Print(w)
	return err
}
