package hush

// Value represents a string contained in the leaf of a Tree.
type Value string

// NewValue returns a new value representing the given plaintext
// string.
func NewValue(v string) Value {
	return Value(v)
}

// Ciphertext returns a version of this value that's been encrypted with
// the given key.
func (v Value) Ciphertext(key []byte) Value {
	return v
}

// Plaintext returns a version of this value that's been decrypted with
// the given key.
func (v Value) Plaintext(key []byte) Value {
	return v
}

// IsEncrypted returns true if this value is encrypted, false if it's
// plaintext.
func (v Value) IsEncrypted() bool {
	return false
}
