package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

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

	for _, tf := range testFiles {
		t.Run(tf, func(t *testing.T) {
			cmd := exec.Command("/bin/bash", tf)

			// we don't support any remote servers in shell tests (those are in bnscli unit tests)
			// BNSCLI_TM_ADDR must be unset
			cmd.Env = append(os.Environ(), "BNSCLI_TM_ADDR=")

			out, err := cmd.Output()
			if err != nil {
				if e, ok := err.(*exec.ExitError); ok {
					t.Logf("Below is the script stderr:\n%s\n\n", string(e.Stderr))
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
