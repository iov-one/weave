package server

import (
	"flag"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
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

type Options struct {
	MinFee coin.Coin
	Debug  bool
	Home   string
	Logger log.Logger
}

func parseFlags(args []string) (string, *Options, error) {
	// parse flagBind and return the result
	var addr string
	var minFeeStr string
	options := &Options{
		MinFee: coin.Coin{},
	}

	startFlags := flag.NewFlagSet("start", flag.ExitOnError)
	startFlags.StringVar(&addr, flagBind, "tcp://localhost:26658", "address server listens on")
	startFlags.StringVar(&minFeeStr, flagMinFee, "0 IOV", "minimal anti-spam fee")
	startFlags.BoolVar(&options.Debug, flagDebug, false, "call stack returned on error")
	err := startFlags.Parse(args)

	if err != nil {
		return addr, options, err
	}

	options.MinFee, err = coin.ParseHumanFormat(minFeeStr)

	return addr, options, err
}

// AppGenerator lets us lazily initialize app, using home dir
// and logger potentially initialized with other flags
type AppGenerator func(*Options) (abci.Application, error)

// StartCmd initializes the application, and
func StartCmd(gen AppGenerator, logger log.Logger, home string, args []string) error {
	addr, options, err := parseFlags(args)
	if err != nil {
		return err
	}
	options.Home = home
	options.Logger = logger

	// Generate the app in the proper dir
	app, err := gen(options)
	if err != nil {
		return err
	}

	logger.Info("Starting ABCI app", "bind", addr)

	svr, err := server.NewServer(addr, "socket", app)
	if err != nil {
		return errors.Wrap(err, "failed to create a listener")
	}

	svr.SetLogger(logger.With("module", "abci-server"))

	done := make(chan bool)
	cleanupCallback := func() {
		// Cleanup
		_ = svr.Stop()
		done <- true
	}

	cmn.TrapSignal(logger, cleanupCallback)

	defer func() {
		if err := recover(); err != nil {
			logger.Error("recovered from panic", "err", err)
			cleanupCallback()
		}
	}()

	err = svr.Start()
	if err != nil {
		return errors.Wrap(err, "failed to start a server")
	}

	// wait forever
	<-done

	return nil
}
