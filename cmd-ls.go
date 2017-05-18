package hush

import "io"

func CmdLs(w io.Writer, tree T, pattern string) {
	tree = tree.filter(pattern)
	tree.Print(w)
}
