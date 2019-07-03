package main

import (
	"bytes"
	"context"
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/iov-one/weave/tmtest"
	"github.com/pmezard/go-difflib/difflib"
)

var goldFl = flag.Bool("gold", false, "If true, write result to golden files instead of comparing with them.")

func TestAll(t *testing.T) {
	ensureBnscliBinary(t)

	testFiles, err := filepath.Glob("./*.test")
	if err != nil {
		t.Fatalf("cannot find test files: %s", err)
	}
	if len(testFiles) == 0 {
		t.Skip("no test files found")
	}

	home, cleanup := tmtest.SetupConfig(t, "testdata")
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	defer tmtest.RunTendermint(ctx, t, home)()
	defer RunBnsd(ctx, t, home)()

	for _, tf := range testFiles {
		t.Run(tf, func(t *testing.T) {
			cmd := exec.Command("/bin/sh", tf)

			// Use host's environment to run the tests. This allows
			// to provide the same setup as when running each
			// script directly.
			// To ensure that all commands are using tendermint
			// mock server, set environment variable to enforce
			// that.
			// Port is set in testdata/config/config.toml under [rpc]laddr
			cmd.Env = append(os.Environ(), "BNSCLI_TM_ADDR=http://localhost:44444")

			out, err := cmd.Output()
			if err != nil {
				if e, ok := err.(*exec.ExitError); ok {
					t.Logf("Below is the cript stderr:\n%s\n\n", string(e.Stderr))
				}
				t.Fatalf("execution failed: %s", err)
			}

			goldFilePath := tf + ".gold"

			if *goldFl {
				if err := ioutil.WriteFile(goldFilePath, out, 0644); err != nil {
					t.Fatalf("cannot write golden file: %s", err)
				}
			}

			want, err := ioutil.ReadFile(goldFilePath)
			if err != nil {
				t.Fatalf("cannot read golden file: %s", err)
			}

			if !bytes.Equal(want, out) {
				diff := difflib.UnifiedDiff{
					A:        difflib.SplitLines(string(want)),
					B:        difflib.SplitLines(string(out)),
					FromFile: "Gold",
					ToFile:   "Current",
					Context:  2,
				}
				text, _ := difflib.GetUnifiedDiffString(diff)
				t.Log(text)
				t.Fatal("unexpected result")
			}
		})
	}
}

func ensureBnscliBinary(t testing.TB) {
	t.Helper()

	if cmd := exec.Command("bnscli", "version"); cmd.Run() != nil {
		t.Skipf(`bnscli binary is not available

You can install bnscli binary by running "make install" in
weave main directory or by directly using Go install command:

  $ go install github.com/iov-one/weave/cmd/bnscli
`)
	}
}

func RunBnsd(ctx context.Context, t *testing.T, home string) (cleanup func()) {
	t.Helper()

	bnsdpath, err := exec.LookPath("bnsd")
	if err != nil {
		if os.Getenv("FORCE_TM_TEST") != "1" {
			t.Skip("Bnsd binary not found. Set FORCE_TM_TEST=1 to fail this test.")
		} else {
			t.Fatalf("Bnsd binary not found. Do not set FORCE_TM_TEST=1 to skip this test.")
		}
	}

	cmd := exec.CommandContext(ctx, bnsdpath, "-home", home, "start")
	// log tendermint output for verbose debugging....
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stdout
	if err := cmd.Start(); err != nil {
		t.Fatalf("Bnsd process failed: %s", err)
	}

	// Give tendermint time to setup.
	time.Sleep(2 * time.Second)
	t.Logf("Running %s pid=%d", bnsdpath, cmd.Process.Pid)

	// Return a cleanup function, that will wait for the tendermint to stop.
	//nolint
	return func() {
		cmd.Process.Kill()
		cmd.Wait()
	}
}
