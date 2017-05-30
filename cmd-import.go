package hush

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// CmdImport reads tab-separated lines from r adding the path-value
// pairs to tree.  Returns a slice of warnings, if any.
//
// This function implements "hush import".
func CmdImport(r io.Reader, tree *Tree) ([]string, error) {
	var n int
	var warnings []string
	warnf := func(format string, args ...interface{}) {
		format = "line " + strconv.Itoa(n) + ": " + format
		msg := fmt.Sprintf(format, args...)
		warnings = append(warnings, msg)
	}

	scanner := bufio.NewScanner(r)
	for n = 1; scanner.Scan(); n++ {
		txt := scanner.Text()
		if txt == "" {
			continue
		}
		parts := strings.SplitN(txt, "\t", 2)
		if len(parts) < 2 {
			warnf("missing tab delimiter")
			continue
		}
		p := NewPath(parts[0])
		if p.IsConfiguration() {
			warnf("skipping configuration path %s", p)
			continue
		}
		val := NewPlaintext([]byte(parts[1]), Private)
		tree.set(p, val)
	}
	err := tree.Save()
	return warnings, errors.Wrap(err, "import")
}
