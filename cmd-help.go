package hush

import (
	"io"
	"os"
)

func CmdHelp(w io.Writer) {
	io.WriteString(os.Stdout, helpMessage)
}

var helpMessage = `NAME
    hush - tiny password manager

SYNOPSIS
    hush [command [arguments]]

INSTALLATION
    go install github.com/mndrix/hush/...

DESCRIPTION
    hush is a password manager with a small, understandable code base.
    Your secrets are stored in a tree with encrypted leaves.  You can
    organize the tree in whatever hierarchy you prefer.

    The hush file (in $HOME/.hush by default) is a plaintext file with
    a simple format reminiscent of YAML.  It's designed to be kept
    under version control, although that's not necessary.  The file
    also contains a cryptographic checksum to avoid unauthorized
    modifications.

COMMANDS
    This section contains a list of commands supported by hush. The
    command name should be the second argument on the command line when
    invoking hush.

    export
        Exports the decrypted contents of your hush file to stdout.
        Each line represents a leaf and the path to that leaf. Each
        line is split into two columns, separated by a tab character.
        The first column is a slash-separated path. The second column
        is the leaf's plaintext.

        See also: import command

    help
        Displays this help text.

    import
        Imports plaintext paths and leaves from stdin into your hush
        file.  The input format is the same as that generated by
        the export command.

        See also: export command

    init
        Initializes a new hush file after prompting the user to
        create a password.  This command must be run before most of
        the other commands can be run.

    ls [pattern]
        Lists all decrypted subtrees matching 'pattern'.  If 'pattern'
        is omitted, lists the entire tree.

        See also: PATTERNS

    rm path [path [path [...]]]
        Removes each path, and its subtrees, from the hush file.

    set path value
        Sets the leaf at 'path' to have 'value'.  The value is stored
        encrypted in the hush file.  The path is not encrypted.

        If value is '-' then the leaf's value is read from stdin.

PATTERNS

    A pattern matches paths within the tree.  A pattern is first split
    on '/' to generate subpatterns.  Each subpattern describes a
    descent one level deeper into the tree.  At each level, a
    subpattern matches all local paths which contain the subpattern as
    a substring.

    For example:

        $ hush ls
        paypal.com:
            personal:
                password: secret
            work:
                password: 123456
        bitpay.com:
            work:
                password: 42 bitcoins

        $ hush ls pay/work
        paypal.com:
            work:
                password: 123456
        bitpay.com:
            work:
                password: 42 bitcoins


ENVIRONMENT VARIABLES
    This section describes environment variables which can be used to
    change the default behavior of hush.

    HUSH_ASKPASS
        When hush needs to request a password, it runs the script
        pointed to by this variable.  The script is invoked with a
        single argument: the text to use in the prompt.  The script's
        stdout is used as the password.

        If you get tired of typing your password repeatedly, you can
        set this variable to a script that caches your password.

        If HUSH_ASKPASS is missing, hush prompts on the user's
        terminal.

    HUSH_FILE
        Set this variable to the absolute path of your hush file.
        The default, if empty, is $HOME/.hush
`
