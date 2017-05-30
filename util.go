package hush

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"golang.org/x/crypto/ssh/terminal"
)

// AskPassword asks the user for a password by presenting the given
// prompt.  Users can point HUSH_ASKPASS environment at a script of
// their choice to change how passwords are collected. In either case,
// the provided password is returned along with any errors.
//
// By default, the prompt is displayed on w and the password is read,
// without echo, from the terminal.
func AskPassword(w io.Writer, prompt string) ([]byte, error) {
	tty, err := os.Open("/dev/tty")
	if err != nil {
		return nil, err
	}

	if askpass := os.Getenv("HUSH_ASKPASS"); askpass != "" {
		cmd := exec.Command(askpass, prompt)
		cmd.Stdin = tty
		cmd.Stderr = os.Stderr
		password, err := cmd.Output()
		if err != nil {
			return nil, err
		}
		return password, nil
	}

	io.WriteString(w, prompt+": ")
	password, err := terminal.ReadPassword(int(tty.Fd()))
	io.WriteString(w, "\n")
	return password, err
}

// Home returns the user's home directory.
func Home() (string, error) {
	home := os.Getenv("HOME")
	if home == "" {
		return "", errors.New("Point $HOME at your home directory")
	}
	return home, nil
}

// HushPath returns the filename of this user's hush file, whether it
// exists or not. If the file doesn't exist, it also returns an error
// for which os.IsNotExist() is true.
//
// Any symlinks along the way are resolved until a concrete file is
// found.
func HushPath() (string, error) {
	if filename := os.Getenv("HUSH_FILE"); filename != "" {
		_, err := os.Stat(filename)
		return filename, err
	}

	home, err := Home()
	if err != nil {
		return "", err
	}
	filename := path.Join(home, ".hush")
	f, err := filepath.EvalSymlinks(filename)
	if err == nil {
		filename = f
	}
	return filename, err
}
