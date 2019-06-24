package main

import (
	"bytes"
	"testing"

	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/escrow"
)

func TestCmdReleaseEscrowHappyPath(t *testing.T) {
	var output bytes.Buffer
	args := []string{
		"-escrow", "5",
		"-amount", "49 DOGE",
	}
	if err := cmdReleaseEscrow(nil, &output, args); err != nil {
		t.Fatalf("cannot create a new release escrow transaction: %s", err)
	}

	tx, _, err := readTx(&output)
	if err != nil {
		t.Fatalf("cannot read created transaction: %s", err)
	}

	txmsg, err := tx.GetMsg()
	if err != nil {
		t.Fatalf("cannot get transaction message: %s", err)
	}
	msg := txmsg.(*escrow.ReleaseMsg)

	assert.Equal(t, sequenceID(5), []byte(msg.EscrowId))
	assert.Equal(t, []*coin.Coin{coin.NewCoinp(49, 0, "DOGE")}, msg.Amount)
}
