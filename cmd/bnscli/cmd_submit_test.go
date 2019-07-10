package main

import (
	"bytes"
	"testing"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/cash"
)

// TestCmdSubmitTxHappyPath will set fees, sign the tx, and submit it... ensuring the
// whole workflow is valid.
func TestCmdSubmitTxHappyPath(t *testing.T) {
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_CashSendMsg{
			CashSendMsg: &cash.SendMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				Source:      fromHex(t, addr),
				Destination: fromHex(t, addr),
				Amount: &coin.Coin{
					Whole:  5,
					Ticker: "IOV",
				},
			},
		},
	}

	var input bytes.Buffer
	if _, err := writeTx(&input, tx); err != nil {
		t.Fatalf("cannot marshal transaction: %s", err)
	}

	var withFee bytes.Buffer
	feeArgs := []string{
		"-tm", tmURL,
	}
	err := cmdWithFee(&input, &withFee, feeArgs)
	assert.Nil(t, err)

	// we must sign it with a key we have
	var signedTx bytes.Buffer
	signArgs := []string{
		"-tm", tmURL,
		"-key", mustCreateFile(t, bytes.NewReader(fromHex(t, privKeyHex))),
	}
	err = cmdSignTransaction(&withFee, &signedTx, signArgs)
	assert.Nil(t, err)

	var output bytes.Buffer
	args := []string{
		"-tm", tmURL,
	}
	if err := cmdSubmitTransaction(&signedTx, &output, args); err != nil {
		t.Fatalf("cannot submit the transaction: %s", err)
	}
}
