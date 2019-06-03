package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// commands is a register of all availables commands that can be executed by
// this program. The name is used to match with the first argument given.
//
// When a cmd function is callend it is given stdin, stdout and command line
// arguments except the program name and this command name. It is the
// responsibility of the command function to parse the arguments. Use os.Stderr
// to write error messages.
//
// TODO - document here how to create a command function. They all must follow
// the same convention. Ie use flag package and provide help information.
var commands = map[string]func(input io.Reader, output io.Writer, args []string) error{
	"transfer-proposal": cmdNewTransferProposal,
	"escrow-proposal":   cmdNewEscrowProposal,
	"sign":              cmdSignTransaction,
	"submit":            cmdSubmitTx,
}

func main() {
	if len(os.Args) == 1 {
		available := make([]string, 0, len(commands))
		for name := range commands {
			available = append(available, name)
		}
		fmt.Fprintf(os.Stderr, "%s is a command line client for the BNSD application.\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Usage: %s <command> [<flags>]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nAvailable commands are:\n\t%s\n", strings.Join(available, "\n\t"))
		fmt.Fprintf(os.Stderr, "Use <command> -help to learn more about each command.\n")
		os.Exit(2)
	}
	run, ok := commands[os.Args[1]]
	if !ok {
		available := make([]string, 0, len(commands))
		for name := range commands {
			available = append(available, name)
		}
		fmt.Fprintf(os.Stderr, "Unknown command %q\n", os.Args[1])
		fmt.Fprintf(os.Stderr, "\nAvailable commands are:\n\t%s\n", strings.Join(available, "\n\t"))
		os.Exit(2)
	}

	// Skip two first arguments. Second argument is the command name that
	// we just consumed.
	if err := run(os.Stdin, os.Stdout, os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
