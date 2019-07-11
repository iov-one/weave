package main

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"testing"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/x/cash"
)

func TestCmdSignTransactionHappyPath(t *testing.T) {
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_CashSendMsg{
			CashSendMsg: &cash.SendMsg{
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
		"-tm", tmURL,
		"-key", mustCreateFile(t, bytes.NewReader(fromHex(t, privKeyHex))),
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
