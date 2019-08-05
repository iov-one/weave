package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/commands"
	"github.com/iov-one/weave/commands/server"
	"github.com/tendermint/tendermint/libs/log"
)

func main() {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout)).
		With("module", "bns")

	defaultHome := filepath.Join(os.ExpandEnv("$HOME"), ".bns")
	varHome := flag.String("home", defaultHome, "directory to store files under")

	flag.CommandLine.Usage = func() { helpMessage(os.Stderr) }

	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "missing command\n")
		helpMessage(os.Stderr)
		os.Exit(2)
	}

	cmd := flag.Arg(0)
	rest := flag.Args()[1:]

	switch cmd {
	case "help":
		helpMessage(os.Stderr)
	case "init":
		exitOnErr(server.InitCmd(bnsd.GenInitOptions, logger, *varHome, rest))
	case "start":
		exitOnErr(server.StartCmd(bnsd.GenerateApp, logger, *varHome, rest))
	case "getblock":
		exitOnErr(server.GetBlockCmd(rest))
	case "retry":
		exitOnErr(server.RetryCmd(bnsd.InlineApp, logger, *varHome, rest))
	case "testgen":
		exitOnErr(commands.TestGenCmd(bnsd.Examples(), rest))
	case "version":
		fmt.Println(weave.Version)
	case "validate":
		exitOnErr(server.ValidateGenesis(bnsd.Initializers(), rest))
	default:
		helpMessage(os.Stderr)
		os.Exit(2)
	}
}

func helpMessage(out io.Writer) {
	fmt.Fprint(out, `bnsd - Blockchain Name Service node")

Available commands:

	getblock    Extract a block from blockchain.db.
	help        Print this message.
	init        Initialize application options in genesis file.
	retry       Run last block again to ensure it produces same result.
	start       Run the ABCI server.
	validate    Parse given genesis file and ensure that defined there state can be loaded.
	version     Print this application version.

Available flags:

	-home string
		Directory to store files under (default "$HOME/.bns")
`)
}

func exitOnErr(err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "%+v\n", err)
	os.Exit(1)
}
