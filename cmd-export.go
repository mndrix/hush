package hush

import "io"

func CmdExport(w io.Writer, t *Tree) error {
	for p, v := range t.tree {
		v, err := v.Plaintext(t.encryptionKey)
		if err != nil {
			return err
		}

		io.WriteString(w, p.AsPattern())
		io.WriteString(w, "\t")
		io.WriteString(w, v.String())
		io.WriteString(w, "\n")
	}

	return nil
}
