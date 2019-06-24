package main

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/x/cash"
)

func TestCmdSignTransactionHappyPath(t *testing.T) {
	tm := newSignTendermintServer(t)
	defer tm.Close()

	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_SendMsg{
			SendMsg: &cash.SendMsg{
				Metadata: &weave.Metadata{Schema: 1},
			},
		},
	}
	var input bytes.Buffer
	if _, err := writeTx(&input, tx); err != nil {
		t.Fatalf("cannot marshal transaction: %s", err)
	}

	var output bytes.Buffer
	args := []string{
		"-tm", tm.URL,
		"-key", mustCreateFile(t, bytes.NewReader(fromHex(t, "d34c1970ae90acf3405f2d99dcaca16d0c7db379f4beafcfdf667b9d69ce350d27f5fb440509dfa79ec883a0510bc9a9614c3d44188881f0c5e402898b4bf3c9"))),
	}
	if err := cmdSignTransaction(&input, &output, args); err != nil {
		t.Fatalf("transaction signing failed: %s", err)
	}

	tx, _, err := readTx(&output)
	if err != nil {
		t.Fatalf("cannot read created transaction: %s", err)
	}

	if n := len(tx.Signatures); n != 1 {
		t.Fatalf("want one signature, got %d", n)
	}
}

func newSignTendermintServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if *logRequestFl {
			b, _ := ioutil.ReadAll(r.Body)
			r.Body = ioutil.NopCloser(bytes.NewReader(b))
			t.Logf("tendermint request: %s %s: %s", r.Method, r.URL.Path, string(b))
		}
		switch {
		case r.Method == "GET" && r.URL.Path == "/genesis":
			io.WriteString(w, `
				{
					"jsonrpc": "2.0",
					"id": "",
					"result": {
						"genesis": {
							"chain_id": "test-chain-ZIYjN0",
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

var logRequestFl = flag.Bool("logrequest", false, "Log all requests send to tendermint mock server. This is useful when writing new test. Use curl to send the same request to a real tendermint node and record the response.")

func mustCreateFile(t testing.TB, r io.Reader) string {
	t.Helper()

	fd, err := ioutil.TempFile("", t.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer fd.Close()
	if _, err := io.Copy(fd, r); err != nil {
		t.Fatal(err)
	}
	if err := fd.Close(); err != nil {
		t.Fatal(err)
	}
	return fd.Name()
}
