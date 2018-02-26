package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tendermint/tmlibs/log"

	"github.com/confio/weave"
	"github.com/confio/weave/commands/server"
	"github.com/confio/weave/std"
)

var (
	flagHome = "home"
	varHome  *string
)

func init() {
	defaultHome := filepath.Join(os.ExpandEnv("$HOME"), ".mycoind")
	varHome = flag.String(flagHome, defaultHome, "directory to store files under")
}

func helpMessage() {
	fmt.Println("mycoind")
	fmt.Println("        MyCoin ABCI Application")
	fmt.Println("")
	fmt.Println("help    Print this message")
	fmt.Println("init    Initialize app options in genesis file")
	fmt.Println("start   Run the abci server")
	fmt.Println("version Print the app version")
	fmt.Println(`
  -home string
        directory to store files under (default "/home/ethan/.mycoind")`)
	// flag.PrintDefaults()
}

func main() {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout)).
		With("module", "mycoin")

	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Println("Missing command:")
		helpMessage()
		os.Exit(1)
	}

	cmd := flag.Arg(0)
	rest := flag.Args()[1:]

	switch cmd {
	case "help":
		helpMessage()
	case "init":
		server.InitCmd(std.GenInitOptions, logger, *varHome, rest)
	case "start":
		server.StartCmd(std.GenerateApp, logger, *varHome, rest)
	case "version":
		fmt.Println(weave.Version())
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		helpMessage()
		os.Exit(1)
	}
}
