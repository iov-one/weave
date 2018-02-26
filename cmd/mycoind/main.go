package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/tendermint/tmlibs/cli"
	"github.com/tendermint/tmlibs/log"

	"github.com/confio/weave"
	"github.com/confio/weave/commands/server"
	"github.com/confio/weave/std"
)

// rootCmd is the entry point for this binary
var (
	rootCmd = &cobra.Command{
		Use:   "mycoind",
		Short: "MyCoin Tendermint Node",
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the app version",
		Run:   func(_ *cobra.Command, _ []string) { fmt.Println(weave.Version()) },
	}
)

func main() {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout)).
		With("module", "mycoin")

	rootCmd.AddCommand(
		server.InitCmd(std.GenInitOptions, logger),
		server.StartCmd(std.GenerateApp, logger),
		versionCmd,
	)

	// prepare and add flags
	rootDir := os.ExpandEnv("$HOME/.mycoind")
	executor := cli.PrepareBaseCmd(rootCmd, "MY", rootDir)
	executor.Execute()
}
