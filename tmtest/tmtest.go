/*

Package tmtest provides helpers for testing using tendermint server.

*/
package tmtest

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

// RunTendermint starts a tendermit process. Returned cleanup function will
// ensure the process has stopped and will block until.
//
// Set FORCE_TM_TEST=1 environment variable to fail the test if the binary is
// not available. This might be desired when running tests by CI.
func RunTendermint(ctx context.Context, t *testing.T, home string) (cleanup func()) {
	t.Helper()

	tmpath, err := exec.LookPath("tendermint")
	if err != nil {
		if os.Getenv("FORCE_TM_TEST") != "1" {
			t.Skip("Tendermint binary not found. Set FORCE_TM_TEST=1 to fail this test.")
		} else {
			t.Fatalf("Tendermint binary not found. Do not set FORCE_TM_TEST=1 to skip this test.")
		}
	}

	cmd := exec.CommandContext(ctx, tmpath, "node", "--home", home)
	// log tendermint output for verbose debugging....
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stdout
	if err := cmd.Start(); err != nil {
		t.Fatalf("Tendermint process failed: %s", err)
	}

	// Give tendermint time to setup.
	time.Sleep(2 * time.Second)
	t.Logf("Running %s pid=%d", tmpath, cmd.Process.Pid)

	// Return a cleanup function, that will wait for the tendermint to stop.
	return func() {
		cmd.Process.Kill()
		cmd.Wait()
	}
}
