package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io"
	"os"
	"os/exec"
	"strings"
)

func init() {
	commands["zsh-completion"] = cmdZshCompletion
}

func cmdZshCompletion(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Generate autocompletion for ZSH. To install it, run:

$ source <( bnscli zsh-completion)

		`)
		fl.PrintDefaults()
	}
	fl.Parse(args)

	var cmds []cmd
	for name := range commands {
		if name == "zsh-completion" {
			// Do not self loop.
			continue
		}
		cmds = append(cmds, cmd{
			Name:   name,
			ShName: strings.Replace(name, "-", "_", -1),
			Args:   extractArguments(name),
		})
	}
	var b bytes.Buffer
	if err := tmpl.Execute(&b, struct{ Cmds []cmd }{Cmds: cmds}); err != nil {
		return err
	}
	b.WriteTo(output)
	return nil
}

type cmd struct {
	Name   string
	ShName string
	Args   []argument
}

// extractArguments returns flag information from given command. This is a
// quite hacky implementation. A separte process is used, because every command
// handler function is allowed to call os.Exit.
//
// This is a "the best result for the least effort" approach implementation.
func extractArguments(cmdName string) []argument {
	var args []argument

	var b bytes.Buffer
	cmd := exec.Command(os.Args[0], cmdName, "-h")
	cmd.Env = os.Environ()
	cmd.Stderr = &b
	cmd.Run()

	rd := bufio.NewReader(&b)
	var arg argument
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			return args
		}
		if arg.Flag != "" {
			arg.Description = strings.TrimSpace(line)
			args = append(args, arg)
			arg = argument{}
		} else if strings.HasPrefix(line, "  -") {
			fields := strings.Fields(line)
			arg.Flag = fields[0]
			//arg.Kind = fields[1]
		}
	}
}

type argument struct {
	Flag        string
	Kind        string
	Description string
}

var tmpl = template.Must(template.New("").Parse(`
compdef _bnscli bnscli

function _bnscli {
    local line

    _arguments -C \
        "-help[Show help information]" \
        "1: :({{range .Cmds}}{{.Name}} {{end}})" \
        "*::arg:->args"

    case $line[1] in
    	{{range .Cmds -}}
        {{.Name}})
            _bnscli_cmd_{{.ShName}}
        ;;
	{{- end}}
    esac
}

{{range .Cmds}}
function _bnscli_cmd_{{.ShName}} {
	_arguments \
	{{range .Args -}}
	"{{.Flag}}[{{.Description}}]" \
	{{end}}
}

{{end}}

`))
