package hush

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"golang.org/x/crypto/pbkdf2"

	"github.com/pkg/errors"

	yaml "gopkg.in/yaml.v2"
)

type Branch struct {
	path Path
	val  *Value
}

type Tree struct {
	branches []Branch
	index    map[Path]int
	free     map[int]bool

	encryptionKey []byte
	macKey        []byte
}

const safePerm = 0600 // rw- --- ---

func LoadTree() (*Tree, error) {
	hushPath, err := HushPath()
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
			return nil, errors.Wrap(err, "can't fix permissions on hush file")
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
		branches: make([]Branch, 0, 3*len(items)),
		index:    make(map[Path]int),
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
			t.set(p, NewEncoded(val, privacy))
		case yaml.MapSlice:
			newT_(val, crumbs, t)
		default:
			panic(fmt.Sprintf("unexpected type: %#v", val))
		}
		crumbs = crumbs[:n] // remove final crumb
	}
}

// implement sort.Interface interface
var _ sort.Interface = &Tree{}

func (t *Tree) Len() int { return len(t.branches) }
func (t *Tree) Less(i, j int) bool {
	// sort free branches to the end
	if t.free[i] {
		return false
	}
	if t.free[j] {
		return true
	}

	return t.branches[i].path < t.branches[j].path
}
func (t *Tree) Swap(i, j int) {
	t.branches[i], t.branches[j] = t.branches[j], t.branches[i]
	t.index[t.branches[i].path] = i
	t.index[t.branches[j].path] = j

	iFree, jFree := t.free[i], t.free[j]
	delete(t.free, i)
	delete(t.free, j)
	if iFree {
		t.free[j] = true
	}
	if jFree {
		t.free[i] = true
	}
}

// Sort sorts the tree in place and defragments any deleted branches.
func (t *Tree) Sort() {
	sort.Stable(t)

	// trim deleted branches
	for i := range t.branches {
		if t.free[i] {
			t.branches = t.branches[0:i]
			break
		}
	}
}

func (t *Tree) mapSlice() yaml.MapSlice {
	var slice yaml.MapSlice
	for _, branch := range t.branches {
		if branch.path.IsChecksum() {
			// skip checksum. it's appended by Save()
			continue
		}
		crumbs := branch.path.AsCrumbs()
		slice = mapSlice_(slice, crumbs, branch.val.String())
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

// Checksum returns a cryptographic message authentication code for this tree.
func (t *Tree) Checksum() []byte {
	if len(t.macKey) < 32 {
		panic("trying to calculate checksum without a MAC key")
	}
	mac := hmac.New(sha256.New, t.macKey)
	for _, branch := range t.branches {
		if branch.path.IsChecksum() {
			continue // don't checksum the checksum
		}
		mac.Write([]byte(branch.path))
		mac.Write([]byte(branch.val.String()))
	}
	sum := mac.Sum(nil)
	return sum
}

// Filter returns a subtree whose branches all match the given
// pattern.
func (t *Tree) Filter(pattern string) *Tree {
	keep := t.Empty()
	for _, branch := range t.branches {
		if matches(branch.path, pattern) {
			keep.set(branch.path, branch.val)
		}
	}
	return keep
}

func isLowercase(s string) bool {
	return s == strings.ToLower(s)
}

func matches(p Path, pattern string) bool {
	ps := p.AsCrumbs()
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
	i, ok := t.index[p]
	if ok {
		return t.branches[i].val, true
	}
	return nil, false
}

func (t *Tree) set(p Path, val *Value) {
	i, ok := t.index[p]
	if ok {
		t.branches[i].val = val
	} else {
		t.branches = append(t.branches, Branch{p, val})
		t.index[p] = len(t.branches) - 1
	}
}

// Delete removes a path and all its descendants from the tree.  Returns
// the number of branches removed.
func (t *Tree) Delete(paths ...Path) int {
	n := 0
	for _, p := range paths {
		for i, branch := range t.branches {
			if p == branch.path || p.HasDescendant(branch.path) {
				t.branches[i] = Branch{}
				delete(t.index, p)
				if t.free == nil {
					t.free = make(map[int]bool)
				}
				t.free[i] = true
				n++
			}
		}
	}
	return n
}

// Encrypt returns a copy of this tree with all leaves encrypted.
func (tree *Tree) Encrypt() *Tree {
	t := tree.Empty()
	for _, branch := range tree.branches {
		p := branch.path
		v := branch.val
		if v == nil {
			panic("trimming didn't remove any empty branch")
		}
		if p.IsPublic() { // don't encrypt public data
			t.set(p, v)
			continue
		}
		if p.IsEncryptionKey() { // value uses different encryption key
			t.set(p, v)
			continue
		}
		t.set(p, v.Ciphertext(t.encryptionKey))
	}
	return t
}

// Encode returns a copy of this tree with all leaves encoded into base64.
func (tree *Tree) Encode() *Tree {
	t := tree.Empty()
	for _, branch := range tree.branches {
		t.set(branch.path, branch.val.Encode())
	}
	return t
}

// Empty returns a copy of this tree with all the keys and values
// removed.  It retains any other data associated with this tree.
func (t *Tree) Empty() *Tree {
	tree := *t // shallow copy
	tree.branches = make([]Branch, 0, len(t.branches))
	tree.index = make(map[Path]int, len(t.branches))
	return &tree
}

// SetPassphrase sets the password that's used for performing
// encryption and decryption.
func (t *Tree) SetPassphrase(password []byte) error {
	p := NewPath("hush-configuration/salt")
	v, ok := t.get(p)
	if !ok {
		return errors.New("hush file missing salt")
	}
	v, err := v.Decode()
	if err != nil {
		return errors.Wrap(err, "decoding salt")
	}
	salt := v.plaintext
	pwKey := stretchPassword(password, salt)

	p = NewPath("hush-configuration/encryption-key")
	v, ok = t.get(p)
	if !ok {
		return errors.New("hush file missing encryption key")
	}
	v, err = v.Plaintext(pwKey)
	if err != nil {
		return fmt.Errorf("incorrect password or corrupted encryption key")
	}
	t.encryptionKey = v.plaintext

	p = NewPath("hush-configuration/mac-key")
	v, ok = t.get(p)
	if !ok {
		return errors.New("hush file missing MAC key")
	}
	v, err = v.Plaintext(pwKey)
	if err != nil {
		return fmt.Errorf("incorrect password or corrupted mac key")
	}
	t.macKey = v.plaintext

	// now that we have a password, we can verify the checksum
	got, ok := t.get(NewPath("hush-tree-checksum"))
	if !ok {
		return errors.New("hush file has no checksum")
	}
	got, err = got.Decode()
	if err != nil {
		return errors.Wrap(err, "decoding checksum")
	}
	expect := t.Checksum()
	if !hmac.Equal(got.plaintext, expect) {
		return errors.New("checksum doesn't match. file modified without hush command?")
	}

	return nil
}

// Decrypt returns a copy of this tree with all leaves decrypted.
func (tree *Tree) Decrypt() *Tree {
	var err error
	t := tree.Empty()
	for _, branch := range tree.branches {
		p := branch.path
		v := branch.val
		if p.IsPublic() { // don't decrypt public data
			t.set(p, v)
			continue
		}
		if p.IsEncryptionKey() { // value uses different encryption key
			t.set(p, v)
			continue
		}
		if p.IsMacKey() { // value uses different encryption key
			t.set(p, v)
			continue
		}
		if p.IsChecksum() { // not encrypted at all
			t.set(p, v)
			continue
		}
		v, err = v.Plaintext(tree.encryptionKey)
		if err != nil {
			panic(fmt.Sprintf("%s: %s", p, err))
		}
		t.set(p, v)
	}
	return t
}

// Print displays a tree for human consumption.
func (tree *Tree) Print(w io.Writer) error {
	tree.Sort()
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
	tree.Sort()
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
	file.Write(data)
	io.WriteString(file, "hush-tree-checksum: ")
	io.WriteString(file, NewPlaintext(tree.Checksum(), Public).Encode().String())
	io.WriteString(file, "\n")
	file.Close()
	if err != nil {
		return errors.Wrap(err, "saving tree")
	}

	// move temporary file over top of permanent file
	hushPath, err := HushPath()
	if os.IsNotExist(err) {
		err = nil // we can create the file
	}
	if err != nil {
		return errors.Wrap(err, "saving tree")
	}
	err = rename(file.Name(), hushPath)
	return errors.Wrap(err, "saving tree")
}

// rename is like os.Rename but it falls back to copy-then-remove if
// the rename() system call fails.
func rename(oldpath, newpath string) error {
	err := os.Rename(oldpath, newpath)
	if err == nil {
		return nil
	}

	// trouble renaming, try to copy then remove instead
	old, err := os.Open(oldpath)
	if err != nil {
		return errors.Wrap(err, "opening source after failed rename")
	}
	defer old.Close()
	new, err := os.Create(newpath)
	if err != nil {
		return errors.Wrap(err, "creating target after failed rename")
	}
	defer new.Close()
	err = os.Chmod(newpath, safePerm)
	if err != nil {
		return errors.Wrap(err, "set permissions after failed rename")
	}
	_, err = io.Copy(new, old)
	if err != nil {
		return errors.Wrap(err, "copying content after failed rename")
	}
	err = os.Remove(oldpath)
	return errors.Wrap(err, "removing source after failed rename")
}

// stretchPassword converts a password and a salt into a
// cryptographically secure key.
func stretchPassword(password, salt []byte) []byte {
	pwKey := pbkdf2.Key(
		password, salt,
		2<<15, // iteration count (about 80ms on modern server)
		32,    // desired key size in bytes
		sha256.New,
	)
	return pwKey
}
