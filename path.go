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

// String returns this path as a slash-separated path string.
func (p Path) String() string {
	return strings.Replace(string(p), "\t", "/", -1)
}

// AsCrumbs returns this path as a slice of separate path components.
func (p Path) AsCrumbs() []string {
	return strings.Split(string(p), "\t")
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

// HasDescendant returns true if p is the parent of d.
func (p Path) HasDescendant(d Path) bool {
	n := len(p)
	if len(d) > n && strings.HasPrefix(string(d), string(p)) {
		return d[n] == '\t'
	}
	return false
}

// IsConfiguration returns true if p is a path describing a portion of
// the tree which belongs to a hush configuration.
func (p Path) IsConfiguration() bool {
	return strings.HasPrefix(string(p), "hush-configuration\t")
}

// IsPublic returns true if p is a path whose value must be publicly
// visible.
func (p Path) IsPublic() bool {
	return p == "hush-configuration\tsalt" ||
		p == "hush-tree-checksum"
}

// IsEncryptionKey returns true if p is the path that stores the
// user's encryption key.
func (p Path) IsEncryptionKey() bool {
	return p == "hush-configuration\tencryption-key"
}

// IsMacKey returns true if p is the path that stores the
// user's MAC key.
func (p Path) IsMacKey() bool {
	return p == "hush-configuration\tmac-key"
}

// IsChecksum returns true if p is the path that stores the
// tree's HMAC.
func (p Path) IsChecksum() bool {
	return p == "hush-tree-checksum"
}
