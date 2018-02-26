package server

import (
	"flag"

	"github.com/pkg/errors"

	"github.com/tendermint/abci/server"
	abci "github.com/tendermint/abci/types"

	cmn "github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/log"
)

var (
	startFlags *flag.FlagSet
	flagBind   = "bind"
	addr       *string
)

func init() {
	startFlags = flag.NewFlagSet("start", flag.ExitOnError)
	addr = startFlags.String(flagBind, "tcp://localhost:46658", "address server listens on")
}

// appGenerator lets us lazily initialize app, using home dir
// and other flags (?) to start
type appGenerator func(string, log.Logger) (abci.Application, error)

// StartCmd initializes the application, and
func StartCmd(gen appGenerator, logger log.Logger, home string, args []string) error {
	err := startFlags.Parse(args)
	if err != nil {
		return err
	}

	// Generate the app in the proper dir
	app, err := gen(home, logger)
	if err != nil {
		return err
	}

	logger.Info("Starting ABCI app", "bind", *addr)

	svr, err := server.NewServer(*addr, "socket", app)
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
