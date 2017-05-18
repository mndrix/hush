package hush

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/pkg/errors"

	yaml "gopkg.in/yaml.v2"
)

type T map[Path]Value

func newT(items yaml.MapSlice) T {
	t := make(T, 3*len(items))
	newT_(items, []string{}, t)
	return t
}

func newT_(items yaml.MapSlice, crumbs []string, t T) {
	n := len(crumbs)
	for _, item := range items {
		key := item.Key.(string)
		crumbs = append(crumbs, key)

		switch val := item.Value.(type) {
		case string:
			p := NewPath(strings.Join(crumbs, "/"))
			t[p] = NewValue(val)
		case yaml.MapSlice:
			newT_(val, crumbs, t)
		default:
			panic(fmt.Sprintf("unexpected type: %#v", val))
		}
		crumbs = crumbs[:n] // remove final crumb
	}
}

func (t T) mapSlice() yaml.MapSlice {
	// sort by key
	kvs := make([][]string, 0, len(t))
	for p, val := range t {
		kvs = append(kvs, []string{string(p), string(val)})
	}
	sort.SliceStable(kvs, func(i, j int) bool {
		return kvs[i][0] < kvs[j][0]
	})

	var slice yaml.MapSlice
	for _, kv := range kvs {
		path := strings.Split(kv[0], "\t")
		slice = mapSlice_(slice, path, kv[1])
	}
	return slice
}

func mapSlice_(slice yaml.MapSlice, path []string, value string) yaml.MapSlice {
	if len(path) == 0 {
		panic("path should never have 0 length")
	}
	if len(path) == 1 {
		return append(slice, yaml.MapItem{
			Key:   path[0],
			Value: value,
		})
	}

	var inner yaml.MapSlice
	if len(slice) == 0 {
		slice = append(slice, yaml.MapItem{Key: path[0]})
	} else {
		final := slice[len(slice)-1]
		if final.Key.(string) == path[0] {
			inner = final.Value.(yaml.MapSlice)
		} else {
			slice = append(slice, yaml.MapItem{Key: path[0]})
		}
	}
	slice[len(slice)-1].Value = mapSlice_(inner, path[1:], value)
	return slice
}

func (t T) filter(pattern string) T {
	keep := make(T)
	for p, val := range t {
		if matches(p, pattern) {
			keep[p] = val
		}
	}
	return keep
}

func (t T) get(p Path) (Value, bool) {
	val, ok := t[p]
	return val, ok
}

func (t T) set(p Path, val Value) {
	t[p] = val
}

func (tree T) encrypt() {
	var key []byte
	for p, v := range tree {
		if !v.IsEncrypted() {
			tree[p] = v.Ciphertext(key)
		}
	}

	/*
		block, err := aes.NewCipher(encryptionKey)
		if err != nil {
			panic(err)
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			panic(err)
		}
		nonce := make([]byte, 12)
		_, err = rand.Read(nonce)
		if err != nil {
			panic(err)
		}

		mapLeaves(tree.items, func(leaf string) string {
			plaintext := make([]byte, 1+len(leaf))
			plaintext[0] = 1 // version number
			copy(plaintext[1:], []byte(leaf))
			ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
			return base64.StdEncoding.EncodeToString(ciphertext)
		})
	*/
}

var encryptionKey = []byte(`0123456789abcdef`)

func (tree T) decrypt() {
	var key []byte
	for p, v := range tree {
		if v.IsEncrypted() {
			tree[p] = v.Plaintext(key)
		}
	}
	/*
		mapLeaves(tree.items, func(leaf string) string {
			data, err := base64.StdEncoding.DecodeString(leaf)
			if err != nil {
				// it must be decrypted already
				return leaf
			}
			if len(data) < 1 {
				panic("too little encrypted data")
			}
			if data[0] != 1 {
				panic("I only understand version 1")
			}
			return string(data[1:])
		})
	*/
}

// Print displays a tree for human consumption.
func (tree T) Print(w io.Writer) error {
	slice := tree.mapSlice()
	data, err := yaml.Marshal(slice)
	if err != nil {
		return errors.Wrap(err, "printing tree")
	}

	_, err = w.Write(data)
	return err
}

// Save stores a tree to disk for permanent, private archival.
func (tree T) Save() error {
	tree.encrypt()
	slice := tree.mapSlice()

	data, err := yaml.Marshal(slice)
	if err != nil {
		return errors.Wrap(err, "saving tree")
	}

	// save to temporary file
	file, err := ioutil.TempFile("", "hush-")
	if err != nil {
		return errors.Wrap(err, "saving tree")
	}
	_, err = file.Write(data)
	file.Close()
	if err != nil {
		return errors.Wrap(err, "saving tree")
	}

	// move temporary file over top of permanent file
	hushPath, err := hushPath()
	if err != nil {
		return errors.Wrap(err, "saving tree")
	}
	err = os.Rename(file.Name(), hushPath)
	return errors.Wrap(err, "saving tree")
}
