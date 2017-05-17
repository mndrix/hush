package hush

// Value represents a string contained in the leaf of a Tree.
type value string

// Ciphertext returns a version of this value that's been encrypted with
// the given key.
func (v value) Ciphertext(key []byte) value {
	return v
}

// Plaintext returns a version of this value that's been decrypted with
// the given key.
func (v value) Plaintext(key []byte) value {
	return v
}

// IsEncrypted returns true if this value is encrypted, false if it's
// plaintext.
func (v value) IsEncrypted() bool {
	return false
}
