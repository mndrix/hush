package hush

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
)

// CmdImport reads import lines from r adding the path-value pairs to
// tree.  Returns a slice of warnings, if any.
//
// This function implements "hush import"
func CmdImport(r io.Reader, tree *Tree) ([]string, error) {
	var warnings []string
	scanner := bufio.NewScanner(r)
	for n := 1; scanner.Scan(); n++ {
		txt := scanner.Text()
		if txt == "" {
			continue
		}
		parts := strings.SplitN(txt, "\t", 2)
		if len(parts) < 2 {
			msg := fmt.Sprintf("line %d missing tab delimiter", n)
			warnings = append(warnings, msg)
			continue
		}
		p := NewPath(parts[0])
		if p.IsConfiguration() {
			continue
		}
		val := NewPlaintext([]byte(parts[1]), Private)
		tree.set(p, val)
	}
	err := tree.Save()
	return nil, errors.Wrap(err, "import")
}
