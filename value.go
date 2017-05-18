package hush

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
)

// Value represents a string contained in the leaf of a Tree.
type Value string

// NewValue returns a new value representing the given plaintext
// string.
func NewValue(v string) Value {
	return Value(v)
}

func gcm(key []byte) cipher.AEAD {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}
	return gcm
}

// this is totally insecure. only here for testing
var nonce = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

// Ciphertext returns a version of this value that's been encrypted with
// the given key.
func (v Value) Ciphertext(key []byte) Value {

	bs := []byte(v)
	plaintext := make([]byte, 1+len(bs))
	plaintext[0] = 1 // version number
	copy(plaintext[1:], bs)
	ciphertext := gcm(key).Seal(nil, nonce, plaintext, nil)
	return Value(base64.StdEncoding.EncodeToString(ciphertext))
}

// Plaintext returns a version of this value that's been decrypted with
// the given key.
func (v Value) Plaintext(key []byte) Value {
	bs, err := base64.StdEncoding.DecodeString(string(v))
	if err != nil {
		// it must be decrypted already
		return v
	}
	bs, err = gcm(key).Open(nil, nonce, bs, nil)
	if err != nil {
		panic("decryption failed: " + err.Error())
	}
	if len(bs) < 1 {
		panic("too little encrypted data")
	}
	if bs[0] != 1 {
		panic(fmt.Sprintf("I only understand version 1, got %d", bs[0]))
	}
	bs = bs[1:] // remove the version number
	return Value(string(bs))
}
