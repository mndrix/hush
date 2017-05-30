package hush

import "io"

// CmdExport writes tree t to w in a tab-separated format suitable for
// use with "hush import" and scripting.
//
// This function implements "hush export".
func CmdExport(w io.Writer, t *Tree) error {
	for _, branch := range t.branches {
		p, v := branch.path, branch.val
		if p.IsConfiguration() || p.IsChecksum() {
			continue
		}
		v, err := v.Plaintext(t.encryptionKey)
		if err != nil {
			return err
		}

		io.WriteString(w, p.String())
		io.WriteString(w, "\t")
		io.WriteString(w, v.String())
		io.WriteString(w, "\n")
	}

	return nil
}
