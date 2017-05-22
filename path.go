package hush

import "strings"

// Path represents the sequence of keys needed to reach a particular
// node with a Tree.
type Path string

// NewPath returns a path representing a slash-separated path provided
// through the UI.
func NewPath(uiPath string) Path {
	return Path(strings.Replace(uiPath, "/", "\t", -1))
}

// AsPattern this path as a slash-separated pattern.
func (p Path) AsPattern() string {
	return strings.Replace(string(p), "\t", "/", -1)
}

// Parent returns a path pointing to the parent of this path.  If the
// path has no parent, returns itself.
func (p Path) Parent() Path {
	if strings.Contains(string(p), "\t") {
		parts := strings.Split(string(p), "\t")
		p = Path(strings.Join(parts[:len(parts)-1], "\t"))
	}
	return p
}

// IsConfiguration returns true if p is a path describing a portion of
// the tree which belongs to a hush configuration.
func (p Path) IsConfiguration() bool {
	return strings.HasPrefix(string(p), "hush-configuration\t")
}

// IsPublic returns true if p is a path whose value must be publicly
// visible.
func (p Path) IsPublic() bool {
	return p == "hush-configuration\tsalt"
}

// IsEncryptionKey returns true if p is the path that stores the
// user's encryption key.
func (p Path) IsEncryptionKey() bool {
	return p == "hush-configuration\tencryption-key"
}
