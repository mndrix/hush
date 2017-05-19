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

	gcm := gcm(key)
	plaintext := []byte(v)
	n := 1 + // version byte
		gcm.NonceSize() + // nonce bytes
		len(plaintext) + // plaintext size +
		gcm.Overhead() // ciphertext overhead
	data := make([]byte, 0, n)
	data = append(data, 1) // version number

	ciphertext := gcm.Seal(nil, nonce, plaintext, data)
	data = append(data, ciphertext...)
	return Value(base64.StdEncoding.EncodeToString(data))
}

// Plaintext returns a version of this value that's been decrypted with
// the given key.
func (v Value) Plaintext(key []byte) Value {
	data, err := base64.StdEncoding.DecodeString(string(v))
	if err != nil {
		panic(err)
	}
	if len(data) < 1 {
		panic("too little data")
	}
	if data[0] != 1 {
		panic(fmt.Sprintf("I only understand version 1, got %d", data[0]))
	}
	ciphertext := data[1:] // remove the version number
	data = data[:1]

	plaintext, err := gcm(key).Open(nil, nonce, ciphertext, data)
	if err != nil {
		panic("decryption failed: " + err.Error())
	}
	return Value(string(plaintext))
}
