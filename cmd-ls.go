package hush

import "io"

func CmdLs(w io.Writer, tree *Tree, pattern string) {
	tree = tree.filter(pattern)
	tree.Print(w)
}
