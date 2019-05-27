package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `Remove non standard declarations from a protobuf file. Usage:
	%s < MYFILE.proto > CLEAN.proto
Where MYFILE is the original protobuf file that should be cleaned.
`, os.Args[0])
	}
	flag.Parse()
	var out bytes.Buffer
	if err := cleanup(os.Stdin, &out); err != nil {
		fail(1, err.Error())
	}
	out.WriteTo(os.Stdout)
}

func fail(code int, tmpl string, args ...interface{}) {
	if !strings.HasSuffix(tmpl, "\n") {
		tmpl += "\n"
	}
	fmt.Fprintf(os.Stderr, tmpl, args...)
	os.Exit(code)

}

func cleanup(in io.Reader, out io.Writer) error {
	rd := bufio.NewReader(in)
	wr := bufio.NewWriter(out)
	defer wr.Flush()

	// Track the scope.
	var (
		// inPluginDecl is set to true if reading content between two
		// [] brackets that defines a plugin content.
		inPluginDecl bool

		// inComment is set to true if reading content of that is a
		// comment.
		inComment bool
	)

	for {
		c, err := rd.ReadByte()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("cannot read: %s", err)
		}

		switch c {
		case '\n':
			// Comments are always single line.
			inComment = false

			if !inPluginDecl {
				if next, err := rd.Peek(2); err == nil && next[0] == '\n' && next[1] == '\n' {
					// Avoid double empty lines.
				} else {
					_ = wr.WriteByte(c)
				}
			}

			if next, err := rd.Peek(12); err == nil {
				line := bytes.TrimSpace(next)

				// Any options declared on the message are ignored.
				if bytes.HasPrefix(line, []byte("option")) {
					rd.ReadString(';')
					if next, _ := rd.Peek(1); next[0] == '\n' {
						rd.ReadByte()
					}
				}

				if bytes.HasPrefix(line, []byte("import")) {
					// This is an import declaration. Read
					// the whole line and rewrite only if
					// it is not a gogoproto import.
					line, err := rd.ReadBytes(';')
					if err != nil {
						if err == io.EOF {
							return nil
						}
						return fmt.Errorf("cannot read: %s", err)
					}
					if !bytes.HasSuffix(line, []byte(`gogoproto/gogo.proto";`)) {
						_, _ = wr.Write(line)
					} else if next, _ := rd.Peek(1); next[0] == '\n' {
						rd.ReadByte()
					}
				}
			}

		case '\\':
			if next, err := rd.Peek(1); err == nil && next[0] == '\\' {
				// The rest of the line is a comment.
				inComment = true
			}
			if inComment || !inPluginDecl {
				_ = wr.WriteByte(c)
			}
		case '[', ']':
			if inComment {
				_ = wr.WriteByte(c)
			} else {
				inPluginDecl = c == '['
			}
		default:
			if inComment || !inPluginDecl {
				_ = wr.WriteByte(c)
			}
		}
	}
}
