package server

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/tendermint/abci/server"
	abci "github.com/tendermint/abci/types"

	cmn "github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/log"
)

const (
	flagAddress = "address"
)

// appGenerator lets us lazily initialize app, using home dir
// and other flags (?) to start
type appGenerator func(string, log.Logger) (abci.Application, error)

// StartCmd runs the service passed in, either
// stand-alone, or in-process with tendermint
func StartCmd(app appGenerator, logger log.Logger) *cobra.Command {
	start := startCmd{
		app:    app,
		logger: logger,
	}
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Run the full node",
		RunE:  start.run,
	}
	// basic flags for abci app
	cmd.Flags().String(flagAddress, "tcp://0.0.0.0:46658", "Listen address")
	return cmd
}

type startCmd struct {
	app    appGenerator
	logger log.Logger
}

func (s startCmd) run(cmd *cobra.Command, args []string) error {
	// Generate the app in the proper dir
	addr := viper.GetString(flagAddress)
	home := viper.GetString("home")
	app, err := s.app(home, s.logger)
	if err != nil {
		return err
	}

	s.logger.Info("Starting ABCI app", "bind", addr)

	svr, err := server.NewServer(addr, "socket", app)
	if err != nil {
		return errors.Errorf("Error creating listener: %v\n", err)
	}
	svr.SetLogger(s.logger.With("module", "abci-server"))
	svr.Start()

	// Wait forever
	cmn.TrapSignal(func() {
		// Cleanup
		svr.Stop()
	})
	return nil
}
