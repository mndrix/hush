package hush

import "io"

func CmdExport(w io.Writer, t *Tree) {
	for p, v := range t.tree {
		io.WriteString(w, p.AsPattern())
		io.WriteString(w, "\t")
		io.WriteString(w, v.Plaintext(t.encryptionKey).String())
		io.WriteString(w, "\n")
	}
}
