package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tendermint/tmlibs/log"

	"github.com/confio/weave/commands"
	"github.com/confio/weave/commands/server"

	bov "github.com/iov-one/bcp-demo"
	"github.com/iov-one/bcp-demo/app"
)

var (
	flagHome = "home"
	varHome  *string
)

func init() {
	defaultHome := filepath.Join(os.ExpandEnv("$HOME"), ".bov")
	varHome = flag.String(flagHome, defaultHome, "directory to store files under")

	flag.CommandLine.Usage = helpMessage
}

func helpMessage() {
	fmt.Println("bov")
	fmt.Println("        Blockchain of Value node")
	fmt.Println("")
	fmt.Println("help    Print this message")
	fmt.Println("init    Initialize app options in genesis file")
	fmt.Println("start   Run the abci server")
	fmt.Println("version Print the app version")
	fmt.Println(`
  -home string
        directory to store files under (default "$HOME/.bov")`)
}

func main() {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout)).
		With("module", "bov")

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
		fmt.Println(bov.Version())
	default:
		err = fmt.Errorf("unknown command: %s", cmd)
	}

	if err != nil {
		fmt.Printf("Error: %+v\n\n", err)
		helpMessage()
		os.Exit(1)
	}
}
