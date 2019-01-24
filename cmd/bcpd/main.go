package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bcpd/app"
	"github.com/iov-one/weave/commands"
	"github.com/iov-one/weave/commands/server"
	"github.com/tendermint/tendermint/libs/log"
)

var (
	flagHome = "home"
	varHome  *string
)

func init() {
	defaultHome := filepath.Join(os.ExpandEnv("$HOME"), ".bcp")
	varHome = flag.String(flagHome, defaultHome, "directory to store files under")

	flag.CommandLine.Usage = helpMessage
}

func helpMessage() {
	fmt.Println("bcp")
	fmt.Println("        Blockchain of Value node")
	fmt.Println("")
	fmt.Println("help    Print this message")
	fmt.Println("init    Initialize app options in genesis file")
	fmt.Println("start   Run the abci server")
	fmt.Println("version Print the app version")
	fmt.Println(`
  -home string
        directory to store files under (default "$HOME/.bcp")`)
}

func main() {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout)).
		With("module", "bcp")

	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Println("Missing command:")
		helpMessage()
		os.Exit(1)
	}

	cmd := flag.Arg(0)
	rest := flag.Args()[1:]

	var err error
	switch cmd {
	case "help":
		helpMessage()
	case "init":
		err = server.InitCmd(app.GenInitOptions, logger, *varHome, rest)
	case "start":
		err = server.StartCmd(app.GenerateApp, logger, *varHome, rest)
	case "testgen":
		err = commands.TestGenCmd(app.Examples(), rest)
	case "version":
		fmt.Println(weave.Version)
	default:
		err = fmt.Errorf("unknown command: %s", cmd)
	}

	if err != nil {
		fmt.Printf("Error: %+v\n\n", err)
		helpMessage()
		os.Exit(1)
	}
}
