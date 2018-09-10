package server

import (
	"flag"

	"github.com/pkg/errors"

	"github.com/tendermint/abci/server"
	abci "github.com/tendermint/abci/types"

	cmn "github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/log"
)

const (
	flagBind  = "bind"
	flagDebug = "debug"
)

func parseFlags(args []string) (string, bool, error) {
	// parse flagBind and return the result
	var addr string
	var debug bool
	startFlags := flag.NewFlagSet("start", flag.ExitOnError)
	startFlags.StringVar(&addr, flagBind, "tcp://localhost:46658", "address server listens on")
	startFlags.BoolVar(&debug, flagDebug, false, "call stack returned with Tx result")
	err := startFlags.Parse(args)
	return addr, debug, err
}

// AppGenerator lets us lazily initialize app, using home dir
// and logger potentially initialized with other flags
type AppGenerator func(string, log.Logger, bool) (abci.Application, error)

// StartCmd initializes the application, and
func StartCmd(gen AppGenerator, logger log.Logger, home string, args []string) error {
	addr, debug, err := parseFlags(args)
	if err != nil {
		return err
	}

	// Generate the app in the proper dir
	app, err := gen(home, logger, debug)
	if err != nil {
		return err
	}

	logger.Info("Starting ABCI app", "bind", addr)

	svr, err := server.NewServer(addr, "socket", app)
	if err != nil {
		return errors.Errorf("Error creating listener: %v\n", err)
	}
	svr.SetLogger(logger.With("module", "abci-server"))
	svr.Start()

	// Wait forever
	cmn.TrapSignal(func() {
		// Cleanup
		svr.Stop()
	})
	return nil
}
