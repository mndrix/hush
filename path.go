package hush

import "strings"

// Path represents the sequence of keys needed to reach a particular
// node with a Tree.
type path string

// NewPath returns a path representing a slash-separated path provided
// through the UI.
func NewPath(uiPath string) path {
	return path(strings.Replace(uiPath, "/", "\t", -1))
}

// AsPattern this path as a slash-separated pattern.
func (p path) AsPattern() string {
	return strings.Replace(string(p), "\t", "/", -1)
}

// Parent returns a path pointing to the parent of this path.  If the
// path has no parent, returns itself.
func (p path) Parent() path {
	if strings.Contains(string(p), "\t") {
		parts := strings.Split(string(p), "\t")
		p = path(strings.Join(parts[:len(parts)-1], "\t"))
	}
	return p
}
