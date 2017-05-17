package hush

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	p "path"
	"sort"
	"strings"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type path string
type value string
type T map[path]value

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
			t[path(strings.Join(crumbs, "\t"))] = value(val)
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

func (t T) decrypt() {} // iterate values decrypting them

func (t T) encrypt() {} // iterate values encrypting them

func (t T) filter(pattern string) T {
	keep := make(T)
	for p, val := range t {
		if matches(p, pattern) {
			keep[p] = val
		}
	}
	return keep
}

func (t T) get(p path) (value, bool) {
	val, ok := t[p]
	return val, ok
}

func (t T) set(p path, val value) {
	t[p] = val
}

type Tree struct {
	items yaml.MapSlice
}

func Main() {
	tree, err := LoadTree()
	if err != nil {
		die("%s\n", err.Error())
	}
	//warn("initial tree = %#v\n", tree)

	if len(os.Args) == 1 {
		err = tree.Print()
		if err != nil {
			die("%s\n", err.Error())
		}
		os.Exit(0)
	}

	switch os.Args[1] {
	case "import": // hush import
		mainImport(tree)
	case "ls": // hush ls foo/bar
		if len(os.Args) < 3 {
			tree.Print()
			return
		}
		mainLs(tree, os.Args[2])
	case "set": // hush set paypal.com/personal/user john@example.com
		mainSetValue(tree)
	default:
		die("Usage: hum ...\n")
	}
}

func mainSetValue(tree T) {
	pattern := os.Args[2]
	val, err := captureValue(os.Args[3])
	if err != nil {
		die("%s\n", err.Error())
	}

	p := path(strings.Replace(pattern, "/", "\t", -1))
	tree.set(p, val)
	tree.Print()
	err = tree.Save()
	if err != nil {
		die("%s\n", err.Error())
	}
}

func mainImport(tree T) {
	scanner := bufio.NewScanner(os.Stdin)
	for n := 1; scanner.Scan(); n++ {
		txt := scanner.Text()
		if txt == "" {
			continue
		}
		parts := strings.SplitN(txt, "\t", 2)
		if len(parts) < 2 {
			warn("line %d missing tab delimiter\n", n)
			continue
		}
		p := path(strings.Replace(parts[0], "/", "\t", -1))
		val := value(parts[1])
		tree.set(p, val)
	}
	tree.Print()
	err := tree.Save()
	if err != nil {
		die("%s\n", err.Error())
	}
}

func mainLs(tree T, pattern string) {
	tree = tree.filter(pattern)
	tree.Print()
}

func isTerminal(file *os.File) bool {
	return terminal.IsTerminal(int(os.Stdin.Fd()))
}

func captureValue(s string) (value, error) {
	if s == "-" {
		if isTerminal(os.Stdout) {
			editor := editor()
			warn("would launch %s to capture value\n", editor)
			return "", nil
		}

		all, err := ioutil.ReadAll(os.Stdin)
		return value(all), err
	}
	return value(s), nil
}

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func warn(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}

func LoadTree() (T, error) {
	hushPath, err := hushPath()
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(hushPath)
	if os.IsNotExist(err) {
		warn("hush file does not exist. assuming an empty one\n")
		return T{}, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "can't stat hush file")
	}
	// TODO reduce file permissions if they're too loose

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
	tree.decrypt()
	return tree, nil
}

func matches(p path, pattern string) bool {
	ps := strings.Split(string(p), "\t")
	patterns := strings.Split(pattern, "/")
	if len(patterns) > len(ps) {
		return false
	}

	for i, pattern := range patterns {
		if !strings.Contains(ps[i], pattern) {
			return false
		}
	}
	return true
}

// Print displays a tree for human consumption.
func (tree T) Print() error {
	slice := tree.mapSlice()
	data, err := yaml.Marshal(slice)
	if err != nil {
		return errors.Wrap(err, "printing tree")
	}

	_, err = os.Stdout.Write(data)
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

func (tree *Tree) sort() {
	sort.SliceStable(tree.items, func(i, j int) bool {
		a := strings.ToLower(tree.items[i].Key.(string))
		b := strings.ToLower(tree.items[j].Key.(string))
		return a < b
	})
}

func (tree *Tree) encrypt() {
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

func (tree *Tree) decrypt() {
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

/*
func mapLeaves(items yaml.MapSlice, f func(string) string) {
	for i := range items {
		item := &items[i]
		val := item.Value
		if items, ok := val.(yaml.MapSlice); ok {
			mapLeaves(items, f)
		} else if str, ok := val.(string); ok {
			item.Value = f(str)
		} else {
			panic("unexpected leaf type")
		}
	}
}
*/

func home() (string, error) {
	home := os.Getenv("HOME")
	if home == "" {
		return "", errors.New("Point $HOME at your home directory")
	}
	return home, nil
}

func hushPath() (string, error) {
	home, err := home()
	if err != nil {
		return "", err
	}
	return p.Join(home, ".hush"), nil
}

var editorVarNames = []string{
	"HUSH_EDITOR",
	"VISUAL",
	"EDITOR",
}

func editor() string {
	for _, varName := range editorVarNames {
		ed := os.Getenv(varName)
		if ed != "" {
			return ed
		}
	}

	ed := "vi"
	warn("environment configures no editor. defaulting to %s", ed)
	return ed
}
