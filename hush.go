package hush

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
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
	fmt.Fprintf(os.Stderr, "%#v\n", tree)

	if len(os.Args) == 3 { // hush paypal.com/personal/user john@example.com
		mainSetValue(tree)
	} else {
		err = tree.Print()
		if err != nil {
			die("%s\n", err.Error())
		}
	}
}

func mainSetValue(tree *Tree) {
	pattern, value := os.Args[1], os.Args[2]
	value, err := captureValue(value)
	if err != nil {
		die("%s\n", err.Error())
	}
	paths, err := tree.Match(pattern)
	if err != nil {
		die("%s\n", err.Error())
	}

	var path []string
	switch len(paths) {
	case 0:
		path = strings.Split(pattern, "/")
	case 1:
		path = paths[0]
	default:
		die("pattern %q matches multiple paths: %s", paths)
	}

	tree.Set(path, value)
	err = tree.Save()
	if err != nil {
		die("%s\n", err.Error())
	}
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
	fmt.Fprintf(os.Stderr, "keys = %#v\n", keys)
	tree := &Tree{keys}
	tree.decrypt()
	return tree, nil
}

func (tree *Tree) Match(pattern string) ([][]string, error) {
	var matches [][]string
	// TODO perform pattern matching
	return matches, nil
}

func (tree *Tree) Get(needle string) (*Tree, bool) {
	for _, item := range tree.items {
		if key, ok := item.Key.(string); ok {
			if key == needle {
				switch v := item.Value.(type) {
				case yaml.MapSlice:
					return &Tree{v}, true
				default:
					die("unexpected value type %#v", item.Value)
				}
			}
		} else {
			die("all keys should be strings not %#v", item.Key)
		}
	}
	return nil, false
}

func (tree *Tree) Set(path []string, val interface{}) {
	if len(path) == 0 {
		die("path should not have 0 length")
	}

	key := path[0]
	t, ok := tree.Get(key)
	if len(path) == 1 {
		if ok {
			t.items[0].Value = val
		} else {
			tree.items = append(tree.items, yaml.MapItem{
				Key:   key,
				Value: val,
			})
		}
	} else {
		if ok {
			t.Set(path[1:], val)
		} else {
			t = &Tree{}
			t.Set(path[1:], val)
			tree.items = append(tree.items, yaml.MapItem{
				Key:   key,
				Value: t.items,
			})
		}
	}
}

// Print displays a tree for human consumption.
func (tree *Tree) Print() error {
	data, err := yaml.Marshal(tree.items)
	if err != nil {
		return errors.Wrap(err, "printing tree")
	}

	_, err = os.Stdout.Write(data)
	return err
}

// Save stores a tree to disk for permanent, private archival.
func (tree *Tree) Save() error {
	tree.encrypt()

	data, err := yaml.Marshal(tree.items)
	if err != nil {
		return errors.Wrap(err, "saving tree")
	}

	// TODO save to temporary file
	// TODO move temporary file over top of permanent file
	_, err = os.Stdout.Write(data)
	return err
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
