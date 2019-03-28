package server

import (
	"flag"
	"fmt"

	"github.com/iov-one/weave/coin"
	"github.com/tendermint/tendermint/abci/server"
	abci "github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"
)

const (
	flagBind   = "bind"
	flagDebug  = "debug"
	flagMinFee = "min_fee"
)

func parseFlags(args []string) (string, bool, coin.Coin, error) {
	// parse flagBind and return the result
	var addr string
	var debug bool
	var minFeeStr string
	minFee := coin.Coin{}

	startFlags := flag.NewFlagSet("start", flag.ExitOnError)
	startFlags.StringVar(&addr, flagBind, "tcp://localhost:46658", "address server listens on")
	startFlags.StringVar(&minFeeStr, flagMinFee, "0 IOV", "minimal anti-spam fee")
	startFlags.BoolVar(&debug, flagDebug, false, "call stack returned on error")
	err := startFlags.Parse(args)

	if err != nil {
		return addr, debug, minFee, err
	}

	err = minFee.UnmarshalJSON([]byte(minFeeStr))

	return addr, debug, minFee, err
}

// AppGenerator lets us lazily initialize app, using home dir
// and logger potentially initialized with other flags
type AppGenerator func(string, log.Logger, bool) (abci.Application, error)

// StartCmd initializes the application, and
func StartCmd(gen AppGenerator, logger log.Logger, home string, args []string) error {
	addr, debug, minFee, err := parseFlags(args)
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
		return fmt.Errorf("Error creating listener: %v\n", err)
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
