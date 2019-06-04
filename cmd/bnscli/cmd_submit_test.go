package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/x/cash"
)

func TestCmdSubmitTxHappyPath(t *testing.T) {
	tm, submitted := newSubmitTendermintServer(t)
	defer tm.Close()

	tx := &app.Tx{
		Sum: &app.Tx_SendMsg{
			SendMsg: &cash.SendMsg{
				Metadata: &weave.Metadata{Schema: 1},
			},
		},
	}
	rawTx, err := tx.Marshal()
	if err != nil {
		t.Fatalf("cannot marshal transaction: %s", err)
	}
	input := bytes.NewReader(rawTx)
	var output bytes.Buffer
	args := []string{
		"-tm", tm.URL,
	}

	if err := cmdSubmitTransaction(input, &output, args); err != nil {
		t.Fatalf("cannot submit the transaction: %s", err)
	}
	if !*submitted {
		t.Fatal("not submitted")
	}
}

func newSubmitTendermintServer(t *testing.T) (*httptest.Server, *bool) {
	t.Helper()

	var submitted bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			submitted = true

		default:
			http.Error(w, "not implemented", http.StatusNotImplemented)
		}
	}))
	return server, &submitted
}
