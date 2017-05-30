package hush

import (
	"io"
	"os"
	"os/exec"

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
