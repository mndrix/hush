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
