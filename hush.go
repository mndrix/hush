package hush

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

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

func mainSetValue(tree *Tree) {
	pattern, value := os.Args[2], os.Args[3]
	value, err := captureValue(value)
	if err != nil {
		die("%s\n", err.Error())
	}

	path := strings.Split(pattern, "/")
	tree.SetPath(path, value)
	tree.Print()
	err = tree.Save()
	if err != nil {
		die("%s\n", err.Error())
	}
}

func mainImport(tree *Tree) {
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
		path := strings.Split(parts[0], "/")
		val := parts[1]
		tree.SetPath(path, val)
	}
	tree.Print()
	err := tree.Save()
	if err != nil {
		die("%s\n", err.Error())
	}
}

func mainLs(tree *Tree, pattern string) {
	tree = tree.Filter(pattern)
	tree.Print()
}

func isTerminal(file *os.File) bool {
	return terminal.IsTerminal(int(os.Stdin.Fd()))
}

func captureValue(value string) (string, error) {
	if value == "-" {
		if isTerminal(os.Stdout) {
			editor := editor()
			warn("would launch %s to capture value\n", editor)
			return "", nil
		}

		all, err := ioutil.ReadAll(os.Stdin)
		return string(all), err
	}
	return value, nil
}

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func warn(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}

func LoadTree() (*Tree, error) {
	hushPath, err := hushPath()
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(hushPath)
	if os.IsNotExist(err) {
		warn("hush file does not exist. assuming an empty one\n")
		return &Tree{}, nil
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
	tree := &Tree{keys}
	tree.decrypt()
	return tree, nil
}

func (tree *Tree) Filter(pattern string) *Tree {
	parts := strings.Split(pattern, "/")
	return tree.filter(parts)
}

func (tree *Tree) filter(parts []string) *Tree {
	if len(parts) == 0 {
		return tree
	}

	var items yaml.MapSlice
	for _, item := range tree.items {
		if s, ok := item.Key.(string); ok {
			if matches(s, parts[0]) {
				if kvs, ok := item.Value.(yaml.MapSlice); ok {
					t := &Tree{items: kvs}
					t = t.filter(parts[1:])
					if len(t.items) > 0 {
						items = append(items, yaml.MapItem{
							Key:   s,
							Value: t.items,
						})
					}
				} else {
					items = append(items, yaml.MapItem{
						Key:   s,
						Value: item.Value,
					})
				}
			}
		} else {
			die("all keys should be strings not %#v\n", item.Key)
		}
	}

	return &Tree{items: items}
}

func matches(s, pattern string) bool {
	return strings.Contains(s, pattern)
}

func (tree *Tree) Get(needle string) (interface{}, bool) {
	for _, item := range tree.items {
		if key, ok := item.Key.(string); ok {
			if key == needle {
				return item.Value, true
			}
		} else {
			die("all keys should be strings not %#v\n", item.Key)
		}
	}
	return nil, false
}

func (tree *Tree) Set(needle string, val interface{}) {
	//warn("Set: %s %s\n", needle, val)
	for i, item := range tree.items {
		if key, ok := item.Key.(string); ok {
			if key == needle {
				tree.items[i].Value = val
				return
			}
		} else {
			die("all keys should be strings not %#v\n", item.Key)
		}
	}

	tree.items = append(tree.items, yaml.MapItem{
		Key:   needle,
		Value: val,
	})
}

func (tree *Tree) SetPath(path []string, val interface{}) {
	//warn("SetPath: %s %s\n", path, val)
	//defer warn("after Set(): %#v\n", tree)
	switch len(path) {
	case 0:
		die("path should not have 0 length")
	case 1:
		tree.Set(path[0], val)
		return
	}

	t := &Tree{}
	key := path[0]
	x, found := tree.Get(key)
	if items, ok := x.(yaml.MapSlice); found && ok {
		//warn("descending into: %s\n", key)
		t.items = items
	} else {
		//warn("creating subtree: %s\n", key)
	}
	t.SetPath(path[1:], val)
	tree.Set(key, t.items)
}

// Print displays a tree for human consumption.
func (tree *Tree) Print() error {
	tree.sort()
	data, err := yaml.Marshal(tree.items)
	if err != nil {
		return errors.Wrap(err, "printing tree")
	}

	_, err = os.Stdout.Write(data)
	return err
}

// Save stores a tree to disk for permanent, private archival.
func (tree *Tree) Save() error {
	tree.sort()
	tree.encrypt()

	data, err := yaml.Marshal(tree.items)
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
	return path.Join(home, ".hush"), nil
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
