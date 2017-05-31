package hush

import "strings"

// Path represents the sequence of keys needed to reach a particular
// node with a Tree.
type Path string

// NewPath returns a path representing the given slash-separated path.
func NewPath(p string) Path {
	return Path(p)
}

// String returns this path in canonical format.
func (p Path) String() string {
	return string(p)
}

// AsCrumbs returns this path as a slice of separate path components.
func (p Path) AsCrumbs() []string {
	return strings.Split(string(p), "/")
}

// Parent returns a path pointing to the parent of this path.  If the
// path has no parent, returns itself.
func (p Path) Parent() Path {
	if strings.Contains(string(p), "/") {
		parts := p.AsCrumbs()
		p = NewPath(strings.Join(parts[:len(parts)-1], "/"))
	}
	return p
}

// HasDescendant returns true if p is the parent of d.
func (p Path) HasDescendant(d Path) bool {
	n := len(p)
	if len(d) > n && strings.HasPrefix(string(d), string(p)) {
		return d[n] == '/'
	}
	return false
}

// IsConfiguration returns true if p is a path describing a portion of
// the tree which belongs to a hush configuration.
func (p Path) IsConfiguration() bool {
	return strings.HasPrefix(string(p), "hush-configuration/")
}

// IsPublic returns true if p is a path whose value must be publicly
// visible.
func (p Path) IsPublic() bool {
	return p == "hush-configuration/salt" ||
		p == "hush-tree-checksum"
}

// IsEncryptionKey returns true if p is the path that stores the
// user's encryption key.
func (p Path) IsEncryptionKey() bool {
	return p == "hush-configuration/encryption-key"
}

// IsMacKey returns true if p is the path that stores the
// user's MAC key.
func (p Path) IsMacKey() bool {
	return p == "hush-configuration/mac-key"
}

// IsChecksum returns true if p is the path that stores the
// tree's HMAC.
func (p Path) IsChecksum() bool {
	return p == "hush-tree-checksum"
}
