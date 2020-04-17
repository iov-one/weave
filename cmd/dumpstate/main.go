package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

var commands = map[string]func(input io.Reader, output io.Writer, args []string) error{
	"calculate-commit-version": nil,
	"generate-json":            cmdGenerateJson,
}

func main() {
	if len(os.Args) == 1 {
		fmt.Fprintf(os.Stderr, "%s is a command line tool for the dumping BNSD application state data\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Usage: %s <command> [<flags>]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nAvailable commands are:\n\t%s\n", strings.Join(availableCmds(), "\n\t"))
		fmt.Fprintf(os.Stderr, "Run '%s <command> -help' to learn more about each command.\n", os.Args[0])
		os.Exit(2)
	}
	run, ok := commands[os.Args[1]]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown command %q\n", os.Args[1])
		fmt.Fprintf(os.Stderr, "\nAvailable commands are:\n\t%s\n", strings.Join(availableCmds(), "\n\t"))
		os.Exit(2)
	}

	// Skip two first arguments. Second argument is the command name that
	// we just consumed.
	if err := run(os.Stdin, os.Stdout, os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func availableCmds() []string {
	available := make([]string, 0, len(commands))
	for name := range commands {
		available = append(available, name)
	}
	sort.Strings(available)
	return available
}

func cmdVersion(in io.Reader, out io.Writer, args []string) error {
	fmt.Fprintln(out, gitHash)
	return nil
}

// gitHash is set during the compilation time.
var gitHash string = "dev"
