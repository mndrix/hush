package hush

import "testing"

var testEncryptionKey []byte

func init() {
	testEncryptionKey = make([]byte, 32)
	for i := range testEncryptionKey {
		testEncryptionKey[i] = byte(i)
	}
}

func TestValueEmpty(t *testing.T) {
	v := NewPlaintext([]byte{}, Private)
	v = v.Ciphertext(testEncryptionKey)
	v, err := v.Plaintext(testEncryptionKey)
	if err != nil {
		t.Errorf("can't decrypt empty string")
		return
	}
}
