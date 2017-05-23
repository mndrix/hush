package hush

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

// Privacy represents the desired level of privacy for a value.
// Either public or private.
type Privacy int8

const (
	_ Privacy = iota // don't use 0. avoid defaulting privacy
	Public
	Private
)

// Value represents a string contained in the leaf of a Tree.
type Value struct {
	privacy    Privacy
	encoded    string // base64 encoded version of the value
	plaintext  []byte
	ciphertext []byte
}

// panic if v is malformed
func (v *Value) validate() {
	switch v.privacy {
	case Public, Private:
		i := 0
		if len(v.encoded) > 0 {
			i++
		}
		if v.plaintext != nil {
			i++
		}
		if v.ciphertext != nil {
			i++
		}
		if i == 1 {
			return
		}
	}

	err := fmt.Errorf("assertion failed. value is malformed: %#v", v)
	panic(err)
}

// NewPlaintext returns a new value representing the given plaintext.
func NewPlaintext(plaintext []byte, privacy Privacy) *Value {
	v := &Value{
		privacy:   privacy,
		plaintext: plaintext,
	}
	v.validate()
	return v
}

// NewCiphertext returns a new value representing the given plaintext.
func NewCiphertext(ciphertext []byte, privacy Privacy) *Value {
	v := &Value{
		privacy:    privacy,
		ciphertext: ciphertext,
	}
	v.validate()
	return v
}

// NewEncoded returns a new value representing an encoded text.  The
// privacy determines whether it's interpreted as an encoded plaintext
// or ciphertext.
func NewEncoded(encoded string, privacy Privacy) *Value {
	v := &Value{
		privacy: privacy,
		encoded: encoded,
	}
	v.validate()
	return v
}

func (v *Value) String() string {
	if v.encoded != "" {
		return v.encoded
	}
	if v.plaintext != nil {
		return string(v.plaintext)
	}
	if v.ciphertext != nil {
		return string(v.ciphertext)
	}
	panic(fmt.Sprintf("String: unexpected state: %#v", v))
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

// Ciphertext returns a version of this value that's been encrypted with
// the given key.
func (v *Value) Ciphertext(key []byte) *Value {
	v, err := v.Decode()
	if err != nil {
		panic("value not encoded correctly: " + err.Error())
	}
	if v.ciphertext != nil {
		return v // already encrypted
	}

	// prepare payload
	gcm := gcm(key)
	plaintext := v.plaintext
	n := 1 + // version byte
		gcm.NonceSize() + // nonce bytes
		len(plaintext) + // plaintext size +
		gcm.Overhead() // ciphertext overhead
	data := make([]byte, 0, n)
	data = append(data, 1) // version number

	// generate nonce
	n = gcm.NonceSize() + 1
	nonce := data[1:n]
	_, err = rand.Read(nonce)
	if err != nil {
		panic("generating nonce: " + err.Error())
	}
	data = data[:n]

	// encrypt
	ciphertext := gcm.Seal(nil, nonce, plaintext, data)
	data = append(data, ciphertext...)
	return NewCiphertext(data, Private)
}

// Plaintext returns a version of this value that's been decrypted with
// the given key.
func (v *Value) Plaintext(key []byte) (*Value, error) {
	v, err := v.Decode()
	if err != nil {
		return nil, err
	}
	if v.plaintext != nil {
		return v, nil // already decrypted
	}
	data := v.ciphertext
	if len(data) < 1 {
		return nil, errors.New("too little data")
	}
	if data[0] != 1 {
		return nil, fmt.Errorf("I only understand version 1, got %d", data[0])
	}

	// extract nonce
	gcm := gcm(key)
	n := gcm.NonceSize() + 1
	nonce := data[1:n]
	ciphertext := data[n:] // remove version and nonce
	data = data[:n]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, data)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %s", err.Error())
	}
	return NewPlaintext(plaintext, Private), nil
}

// Encode returns a version of this value that's been wrapped in
// base64 encoding.  It's a noop if the value has already been
// encoded.
func (v *Value) Encode() *Value {
	if v.encoded != "" {
		return v // value is already encoded
	}
	if v.privacy == Public && v.plaintext != nil {
		v = &Value{
			privacy: Public,
			encoded: base64.StdEncoding.EncodeToString(v.plaintext),
		}
		v.validate()
		return v
	}
	if v.privacy == Private && v.ciphertext != nil {
		v = &Value{
			privacy: Private,
			encoded: base64.StdEncoding.EncodeToString(v.ciphertext),
		}
		v.validate()
		return v
	}
	panic(fmt.Sprintf("Encode: unexpected state: %#v", v))
}

// Decode returns a version of this value that's had all base64
// encoding removed. It's a noop if the value has already been
// decoded.
func (v *Value) Decode() (*Value, error) {
	if v.encoded == "" {
		return v, nil // value is already decoded
	}
	decoded, err := base64.StdEncoding.DecodeString(v.encoded)
	if err != nil {
		return nil, err
	}
	if v.privacy == Public {
		v = &Value{
			privacy:   Public,
			plaintext: decoded,
		}
		v.validate()
		return v, nil
	}
	if v.privacy == Private {
		v = &Value{
			privacy:    Private,
			ciphertext: decoded,
		}
		v.validate()
		return v, nil
	}
	panic("Decode: unexpected state")
}
