package hush

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
)

func CmdInit(w io.Writer, input *os.File) error {
	// make sure hush file doesn't exist yet
	hushFilename, err := hushPath()
	if !os.IsNotExist(err) {
		return fmt.Errorf(
			"A hush file already exists at %s\nYou don't have to run init again",
			hushFilename,
		)
	}

	// prompt for passwords
	io.WriteString(w, "Preparing to initialize your hush file. Please provide\n")
	io.WriteString(w, "and verify a password to use for encryption.\n")
	io.WriteString(w, "\n")
	password, err := readPassword(w, input, "Password: ")
	if err != nil {
		return err
	}
	verify, err := readPassword(w, input, "Verify password: ")
	if err != nil {
		return err
	}
	if !bytes.Equal(password, verify) {
		return errors.New("Passwords don't match")
	}

	// generate keys
	encryptionKey := make([]byte, 32) // 256-bit key for AES
	_, err = rand.Read(encryptionKey)
	if err != nil {
		return err
	}
	salt := make([]byte, 16) // double the RFC8018 minimum
	_, err = rand.Read(salt)
	if err != nil {
		return err
	}
	pwKey := stretchPassword(password, salt)

	t := newT(nil)
	p := NewPath("hush-configuration/salt")
	v := NewPlaintext(salt, Public)
	t.set(p, v)
	p = NewPath("hush-configuration/encryption-key")
	v = NewPlaintext(encryptionKey, Private)
	v = v.Ciphertext(pwKey)
	t.set(p, v)
	err = t.Save()
	if err != nil {
		die("%s", err)
	}

	fmt.Fprintf(w, "Hush file created at %s\n", hushFilename)
	return nil
}
