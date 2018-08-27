package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tmlibs/log"

	"github.com/iov-one/weave/commands/server"
	"github.com/iov-one/weave/cmd/bcpd/app"
)

func TestStartStandAlone(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ABCI stand-alone test")
	}

	home := setupConfig(t)
	defer os.RemoveAll(home)

	logger := log.NewNopLogger()

	err := server.InitCmd(app.GenInitOptions, logger, home, nil)
	require.NoError(t, err)

	// set up app and start up
	args := []string{"-bind", "localhost:11122"}
	runStart := func() error {
		return server.StartCmd(app.GenerateApp, logger, home, args)
	}
	timeout := time.Duration(2) * time.Second
	err = runOrTimeout(runStart, timeout)
	require.NoError(t, err)
}

func TestStartWithTendermint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Tendermint integration test")
	}

	home := setupConfig(t)
	defer os.RemoveAll(home)

	const runTime = 5     // how many seconds to run both processes
	const startupTime = 2 // how many seconds to let tendermint startup

	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout)).
		With("module", "test-cmd")
	err := server.InitCmd(app.GenInitOptions, logger, home, nil)
	require.NoError(t, err)

	// start up tendermint process in the background...
	// this will block 2 seconds and ensure tendermint lives
	// at least 3 seconds after we run StartCmd
	runTendermint(t, home, startupTime, runTime)

	// set up app and start up
	args := []string{"-bind", "localhost:46658"}
	runStart := func() error {
		return server.StartCmd(app.GenerateApp, logger, home, args)
	}
	timeout := time.Duration(runTime+1) * time.Second
	err = runOrTimeout(runStart, timeout)
	require.NoError(t, err)

	// give time for tendermint to shut down
	fmt.Println(">>> Waiting for tendermint to shut down")
	time.Sleep(time.Second + time.Second)
}

func runOrTimeout(cmd func() error, timeout time.Duration) error {
	done := make(chan error)
	go func(out chan<- error) {
		// we assume cmd should block (RunForever)
		err := cmd()
		if err != nil {
			out <- err
		}
		out <- fmt.Errorf("start died for unknown reasons")
	}(done)

	timer := time.NewTimer(timeout)
	select {
	case err := <-done:
		return err
	case <-timer.C:
		return nil
	}
}

// wait startupDelay before returning
// fails if tendermint doesn't run at least startupDelay + timeout
func runTendermint(t *testing.T, home string, startupDelay, timeout int64) {
	tmBin := filepath.Join(
		os.ExpandEnv("$GOPATH"), "bin", "tendermint")

	// runTM should take longer than startupDelay and timeout...
	runTm := func() error {
		killTime := time.Duration(startupDelay + timeout + 2)

		cmd := exec.Command(tmBin, "node", "--home", home)
		fmt.Println(">>> Running tendermint...")

		// log tendermint output for verbose debugging....
		// cmd.Stdout = os.Stdout
		// cmd.Stderr = os.Stdout

		// run it
		err := cmd.Start()
		if err != nil {
			return err
		}

		// after the given time, kill the process...
		time.Sleep(killTime * time.Second)
		fmt.Println("Killing tendermint...")
		cmd.Process.Kill()
		return nil
	}

	runTime := time.Duration(timeout + startupDelay)
	go func(t *testing.T) {
		err := runOrTimeout(runTm, runTime*time.Second)
		assert.NoError(t, err)
	}(t)

	time.Sleep(time.Duration(startupDelay) * time.Second)
}
