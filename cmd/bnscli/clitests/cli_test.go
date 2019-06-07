package main

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
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

	tm := fakeTendermintServer(t)
	defer tm.Close()

	for _, tf := range testFiles {
		t.Run(tf, func(t *testing.T) {
			cmd := exec.Command("/bin/sh", tf)

			// Use host's environment to run the tests. This allows
			// to provide the same setup as when running each
			// script directly.
			// To ensure that all commands are using tendermint
			// mock server, set environment variable to enforce
			// that.
			cmd.Env = append(os.Environ(), "BNSCLI_TM_ADDR="+tm.URL)

			stderr, err := cmd.StderrPipe()
			if err != nil {
				t.Fatalf("cannot get commands stderr: %s", err)
			}

			out, err := cmd.Output()
			if err != nil {
				if b, _ := ioutil.ReadAll(stderr); len(b) != 0 {
					t.Logf("\nSTDERR\n%s\n\n", string(b))
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
				t.Logf("want: %s", string(want))
				t.Logf(" got: %s", string(out))
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

// fakeTendermintServer creates an HTTP server that acts similar to the
// tendermint HTTP server. It does not maintain any state or return fully
// correct responses but its behaviour is good enough to fool bnscli.
func fakeTendermintServer(t testing.TB) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/genesis":
			io.WriteString(w, `
				{
					"jsonrpc": "2.0",
					"id": "",
					"result": {
						"genesis": {
							"chain_id": "bnscli-test-fake-chain",
							"validators": [],
							"app_state": {}
						}
					}
				}
			`)
		case r.Method == "POST" && r.URL.Path == "/":
			// This is an RPC call - response always depends on the
			// submitted content. For our tests it does not matter
			// that much what is returned.
			io.WriteString(w, `
				{
					"jsonrpc": "2.0",
					"id": "jsonrpc-client",
					"result": {"response": {"height": "12345"}}
				}
			`)

		default:
			http.Error(w, "not implemented", http.StatusNotImplemented)
		}
	}))
}
