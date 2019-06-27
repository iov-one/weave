package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/msgfee"
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
		case r.Method == "GET" && r.URL.Path == "/abci_query":
			// Handle all data queries. Values are JSON encoded.

			switch r.URL.Query().Get("data") {
			case `"_c:cash"`:
				w.Header().Set("content-type", "application/json")
				io.WriteString(w, tmGconfResponse(t, &cash.Configuration{
					MinimalFee: coin.NewCoin(11, 0, "BTC"),
				}))
			case `"msgfee:cash/send"`:
				w.Header().Set("content-type", "application/json")
				io.WriteString(w, tmGconfResponse(t, &msgfee.MsgFee{
					MsgPath: "cash/send",
					Fee:     coin.NewCoin(17, 0, "BTC"),
				}))
			default:
				t.Logf("unexpected ABCI query request: %q", r.URL)
				http.Error(w, "not implemented", http.StatusNotImplemented)
			}
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

// tmGconfResponse returns a tenderming HTTP response for a configuration query.
// Returned response does not contain "key" or "height" information.
func tmGconfResponse(t testing.TB, conf interface{ Marshal() ([]byte, error) }) string {
	raw, err := conf.Marshal()
	if err != nil {
		t.Fatalf("cannot marshal configuration: %s", err)
	}
	baseConf := base64.StdEncoding.EncodeToString(raw)
	return `{
	  "jsonrpc": "2.0",
	  "id": "",
	  "result": {
	    "response": {
	      "value": "` + baseConf + `"
	    }
	  }
	}`
}
