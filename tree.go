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

type Tree struct {
	tree map[Path]*Value

	encryptionKey []byte
}

const safePerm = 0600 // rw- --- ---

func LoadTree() (*Tree, error) {
	hushPath, err := hushPath()
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(hushPath)
	if os.IsNotExist(err) {
		warn("hush file does not exist. assuming an empty one")
		return &Tree{}, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "can't stat hush file")
	}
	if (stat.Mode() & os.ModePerm) != safePerm {
		warn("hush file has loose permissions. fixing.")
		err := os.Chmod(hushPath, safePerm)
		if err != nil {
			die("couldn't fix permissions on hush file")
		}
	}

	file, err := os.Open(hushPath)
	if err != nil {
		return nil, errors.Wrap(err, "opening hush file")
	}
	hushData, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, errors.Wrap(err, "can't read hush file")
	}

	keys := make(yaml.MapSlice, 0)
	err = yaml.Unmarshal(hushData, &keys)
	if err != nil {
		return nil, errors.Wrap(err, "can't parse hush file")
	}
	tree := newT(keys)
	return tree, nil
}

func newT(items yaml.MapSlice) *Tree {
	t := &Tree{
		tree: make(map[Path]*Value, 3*len(items)),
	}
	newT_(items, []string{}, t)
	return t
}

func newT_(items yaml.MapSlice, crumbs []string, t *Tree) {
	n := len(crumbs)
	for _, item := range items {
		key := item.Key.(string)
		crumbs = append(crumbs, key)

		switch val := item.Value.(type) {
		case string:
			p := NewPath(strings.Join(crumbs, "/"))
			privacy := Private
			if p.IsPublic() {
				privacy = Public
			}
			t.tree[p] = NewEncoded(val, privacy)
		case yaml.MapSlice:
			newT_(val, crumbs, t)
		default:
			panic(fmt.Sprintf("unexpected type: %#v", val))
		}
		crumbs = crumbs[:n] // remove final crumb
	}
}

func (t *Tree) mapSlice() yaml.MapSlice {
	// sort by key
	kvs := make([][]string, 0, len(t.tree))
	for p, val := range t.tree {
		kvs = append(kvs, []string{string(p), val.String()})
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

func (t *Tree) filter(pattern string) *Tree {
	keep := t.Empty()
	for p, val := range t.tree {
		if matches(p, pattern) {
			keep.tree[p] = val
		}
	}
	return keep
}

func isLowercase(s string) bool {
	return s == strings.ToLower(s)
}

func matches(p Path, pattern string) bool {
	ps := strings.Split(string(p), "\t")
	patterns := strings.Split(pattern, "/")
	if len(patterns) > len(ps) {
		return false
	}

	ignoreCase := isLowercase(pattern)
	for i, pattern := range patterns {
		haystack := ps[i]
		if ignoreCase {
			haystack = strings.ToLower(haystack)
		}
		if !strings.Contains(haystack, pattern) {
			return false
		}
	}
	return true
}

func (t *Tree) get(p Path) (*Value, bool) {
	val, ok := t.tree[p]
	return val, ok
}

func (t *Tree) set(p Path, val *Value) {
	t.tree[p] = val
}

// Encrypt returns a copy of this tree with all leaves encrypted.
func (tree *Tree) Encrypt() *Tree {
	t := tree.Empty()
	for p, v := range tree.tree {
		if p.IsPublic() { // don't encrypt public data
			t.tree[p] = v
			continue
		}
		if p.IsEncryptionKey() { // value uses different encryption key
			t.tree[p] = v
			continue
		}
		t.tree[p] = v.Ciphertext(t.encryptionKey)
	}
	return t
}

// Encode returns a copy of this tree with all leaves encoded into base64.
func (tree *Tree) Encode() *Tree {
	t := tree.Empty()
	for p, v := range tree.tree {
		t.tree[p] = v.Encode()
	}
	return t
}

// Empty returns a copy of this tree with all the keys and values
// removed.  It retains any other data associated with this tree.
func (t *Tree) Empty() *Tree {
	tree := &Tree{
		tree:          make(map[Path]*Value, len(t.tree)),
		encryptionKey: t.encryptionKey,
	}
	return tree
}

// SetPassphrase sets the password that's used for performing
// encryption and decryption.
func (t *Tree) SetPassphrase(password []byte) {
	t.encryptionKey = []byte(`0123456789abcdef`)
}

// Decrypt returns a copy of this tree with all leaves decrypted.
func (tree *Tree) Decrypt() *Tree {
	t := tree.Empty()
	for p, v := range tree.tree {
		if p.IsPublic() { // don't decrypt public data
			t.tree[p] = v
			continue
		}
		if p.IsEncryptionKey() { // value uses different encryption key
			t.tree[p] = v
			continue
		}
		t.tree[p] = v.Plaintext(tree.encryptionKey)
	}
	return t
}

// Print displays a tree for human consumption.
func (tree *Tree) Print(w io.Writer) error {
	tree = tree.Decrypt()
	slice := tree.mapSlice()
	data, err := yaml.Marshal(slice)
	if err != nil {
		return errors.Wrap(err, "printing tree")
	}

	_, err = w.Write(data)
	return err
}

// Save stores a tree to disk for permanent, private archival.
func (tree *Tree) Save() error {
	tree = tree.Encrypt().Encode()
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
	err = os.Chmod(file.Name(), safePerm)
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
	if os.IsNotExist(err) {
		err = nil // we can create the file
	}
	if err != nil {
		return errors.Wrap(err, "saving tree")
	}
	err = os.Rename(file.Name(), hushPath)
	return errors.Wrap(err, "saving tree")
}
