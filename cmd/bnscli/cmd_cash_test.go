package main

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/cash"
)

func TestCmdSendTokensHappyPath(t *testing.T) {
	var output bytes.Buffer
	args := []string{
		"-src", "b1ca7e78f74423ae01da3b51e676934d9105f282",
		"-dst", "E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0",
		"-amount", "5 DOGE",
		"-memo", "a memo",
	}
	if err := cmdSendTokens(nil, &output, args); err != nil {
		t.Fatalf("cannot create a new token transfer transaction: %s", err)
	}

	tx, _, err := readTx(&output)
	if err != nil {
		t.Fatalf("cannot unmarshal created transaction: %s", err)
	}

	txmsg, err := tx.GetMsg()
	if err != nil {
		t.Fatalf("cannot get transaction message: %s", err)
	}
	msg := txmsg.(*cash.SendMsg)

	assert.Equal(t, fromHex(t, "b1ca7e78f74423ae01da3b51e676934d9105f282"), []byte(msg.Src))
	assert.Equal(t, fromHex(t, "E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0"), []byte(msg.Dest))
	assert.Equal(t, "a memo", msg.Memo)
	assert.Equal(t, coin.NewCoinp(5, 0, "DOGE"), msg.Amount)
}

func TestCmdWithFeeHappyPath(t *testing.T) {
	sendMsg := &cash.SendMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Src:      fromHex(t, "b1ca7e78f74423ae01da3b51e676934d9105f282"),
		Dest:     fromHex(t, "E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0"),
		Amount:   coin.NewCoinp(5, 0, "DOGE"),
		Memo:     "a memo",
	}
	sendTx := &app.Tx{
		Sum: &app.Tx_SendMsg{
			SendMsg: sendMsg,
		},
	}
	var input bytes.Buffer
	if _, err := writeTx(&input, sendTx); err != nil {
		t.Fatalf("cannot serialize transaction: %s", err)
	}

	var output bytes.Buffer
	args := []string{
		"-payer", "b1ca7e78f74423ae01da3b51e676934d9105f282",
		"-amount", "5 DOGE",
	}
	if err := cmdWithFee(&input, &output, args); err != nil {
		t.Fatalf("cannot attach a fee to transaction: %s", err)
	}

	tx, _, err := readTx(&output)
	if err != nil {
		t.Fatalf("cannot unmarshal created transaction: %s", err)
	}
	assert.Equal(t, fromHex(t, "b1ca7e78f74423ae01da3b51e676934d9105f282"), []byte(tx.Fees.Payer))
	assert.Equal(t, coin.NewCoinp(5, 0, "DOGE"), tx.Fees.Fees)

	txmsg, err := tx.GetMsg()
	if err != nil {
		t.Fatalf("cannot get transaction message: %s", err)
	}
	// Message must be unmodified.
	assert.Equal(t, sendMsg, txmsg)
}

func TestCmdWithFeeHappyPathDefaultAmount(t *testing.T) {
	sendMsg := &cash.SendMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Src:      fromHex(t, "b1ca7e78f74423ae01da3b51e676934d9105f282"),
		Dest:     fromHex(t, "E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0"),
		Amount:   coin.NewCoinp(5, 0, "DOGE"),
		Memo:     "a memo",
	}
	sendTx := &app.Tx{
		Sum: &app.Tx_SendMsg{
			SendMsg: sendMsg,
		},
	}
	var input bytes.Buffer
	if _, err := writeTx(&input, sendTx); err != nil {
		t.Fatalf("cannot serialize transaction: %s", err)
	}

	conf := cash.Configuration{
		MinimalFee: coin.NewCoin(4, 0, "BTC"),
	}
	tm := newCashConfTendermintServer(t, conf)
	defer tm.Close()

	var output bytes.Buffer
	args := []string{
		// Instead of providing an amount, rely on what is configured
		// for the network.
		"-tm", tm.URL,
	}
	if err := cmdWithFee(&input, &output, args); err != nil {
		t.Fatalf("cannot attach a fee to transaction: %s", err)
	}

	tx, _, err := readTx(&output)
	if err != nil {
		t.Fatalf("cannot unmarshal created transaction: %s", err)
	}
	assert.Equal(t, []byte(nil), tx.Fees.Payer)
	assert.Equal(t, &conf.MinimalFee, tx.Fees.Fees)

	txmsg, err := tx.GetMsg()
	if err != nil {
		t.Fatalf("cannot get transaction message: %s", err)
	}
	// Message must be unmodified.
	assert.Equal(t, sendMsg, txmsg)
}

// newCashConfTendermintServer returns an HTTP server that can respond to an
// HTTP "/abci_query" request with given configuration.
func newCashConfTendermintServer(t *testing.T, conf cash.Configuration) *httptest.Server {
	t.Helper()

	raw, err := conf.Marshal()
	if err != nil {
		t.Fatalf("cannot marshal configuration: %s", err)
	}
	baseConf := base64.StdEncoding.EncodeToString(raw)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/abci_query" {
			t.Fatalf("unexpected tendermint request: %s", r.URL)
		}
		io.WriteString(w, `
			{
			  "jsonrpc": "2.0",
			  "id": "",
			  "result": {
			    "response": {
			      "key": "CgdfYzpjYXNo",
			      "value": "`+baseConf+`",
			      "height": "1119"
			    }
			  }
			}
		`)
	}))
}
