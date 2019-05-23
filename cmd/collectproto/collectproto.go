package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	// Keep track of those file paths that were already processed to avoid
	// including the same declaration more than once.
	processed := make(map[string]struct{})

	var out bytes.Buffer

	// Syntax declaration is never rewritten. Initialize combined file with
	// one as it is required.
	fmt.Fprintln(&out, `syntax = "proto3";`)

	// Stack of all files that are to be processed.
	protofiles := append([]string{}, os.Args[1:]...)

	for len(protofiles) != 0 {
		// Pop one.
		path := protofiles[0]
		protofiles = protofiles[1:]
		if strings.HasPrefix(path, "github.com") {
			if strings.HasPrefix(path, iovGitHubPrefix) {
				path = path[len(iovGitHubPrefix):]
			} else {
				// Ignore all github imports that are not IOV.
				// This is for example a gogoproto import.
				continue
			}
		}
		if _, ok := processed[path]; ok {
			continue
		}

		fd, err := os.Open(path)
		if err != nil {
			fail(2, "cannot open file: %s", err)
		}

		fmt.Fprintf(&out, "\n\n// definitions from %s\n", iovGitHubPrefix+path)

		imports, err := collect(fd, &out)
		fd.Close()
		if err != nil {
			fail(2, "format: %s", err.Error())
		}

		for _, i := range imports {
			if _, ok := processed[i]; !ok {
				protofiles = append(protofiles, i)
			}
		}
	}

	out.WriteTo(os.Stdout)
}

const iovGitHubPrefix = "github.com/iov-one/weave/"

func fail(code int, tmpl string, args ...interface{}) {
	if !strings.HasSuffix(tmpl, "\n") {
		tmpl += "\n"
	}
	fmt.Fprintf(os.Stderr, tmpl, args...)
	os.Exit(code)

}

func collect(in io.Reader, out io.Writer) ([]string, error) {
	rd := bufio.NewReader(in)
	wr := bufio.NewWriter(out)
	defer wr.Flush()

	var imports []string

	// Track the scope.
	var (
		// inPluginDecl is set to true if reading content between two
		// [] brackets that defines a plugin content.
		inPluginDecl bool

		// inComment is set to true if reading content of that is a
		// comment.
		inComment bool
	)

	shouldWriteChar := func() bool {
		return inComment || !inPluginDecl
	}

	// Pretend the first character read was a new line for easier parsing.
	c := byte('\n')
	for {
		switch c {
		case '\n':
			// Comments are always single line.
			inComment = false

			// Package declarations are not rewritten.
			if next, err := rd.Peek(8); err == nil && bytes.Equal(next, []byte("package ")) {
				rd.ReadString('\n')
				continue
			}
			// Syntax declaration are not rewritten as it should be provided only once.
			if next, err := rd.Peek(7); err == nil && bytes.Equal(next, []byte("syntax ")) {
				rd.ReadString('\n')
				continue
			}

			// Any options declared on the message are ignored.
			if next, err := rd.Peek(12); err == nil && bytes.HasPrefix(bytes.TrimLeft(next, " \t"), []byte("option")) {
				rd.ReadString('\n')
				continue
			}

			// Import declarations are not rewritten but parsed and returned from this function.
			if next, err := rd.Peek(7); err == nil && bytes.Equal(next, []byte("import ")) {
				line, err := rd.ReadString(';')
				if err != nil {
					return nil, fmt.Errorf("cannot read import line: %s", err)
				}

				// Discard new line character to avoid empty lines.
				if next, err := rd.Peek(1); err == nil && next[0] == '\n' {
					_, _ = rd.ReadByte()
				}

				// Thim space, remove "import" and ";"
				line = strings.TrimSpace(line[6 : len(line)-1])
				if line[0] != '"' || line[len(line)-1] != '"' {
					return nil, fmt.Errorf("unexpected import declaration line: %q", line)
				}
				path := line[1 : len(line)-1]
				imports = append(imports, path)
				continue
			}

			if shouldWriteChar() {
				if next, err := rd.Peek(2); err == nil && next[0] == '\n' && next[1] == '\n' {
					// Avoid double empty lines.
				} else {
					_ = wr.WriteByte(c)
				}
			}
		case '\\':
			if next, err := rd.Peek(1); err == nil && next[0] == '\\' {
				// The rest of the line is a comment.
				inComment = true
			}
			if shouldWriteChar() {
				_ = wr.WriteByte(c)
			}
		case '[', ']':
			if inComment {
				_ = wr.WriteByte(c)
			} else {
				inPluginDecl = c == '['
			}
		default:
			if shouldWriteChar() {
				_ = wr.WriteByte(c)
			}
		}

		var err error
		c, err = rd.ReadByte()
		if err != nil {
			if err == io.EOF {
				return imports, nil
			}
			return imports, fmt.Errorf("cannot read: %s", err)
		}
	}
}
