package main

import (
	"bytes"
	"testing"

	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/escrow"
)

func TestCmdReleaseEscrowHappyPath(t *testing.T) {
	var output bytes.Buffer
	args := []string{
		"-escrow", "E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0",
		"-amount", "49 DOGE",
	}
	if err := cmdReleaseEscrow(nil, &output, args); err != nil {
		t.Fatalf("cannot create a new release escrow transaction: %s", err)
	}

	var tx app.Tx
	if err := tx.Unmarshal(output.Bytes()); err != nil {
		t.Fatalf("cannot unmarshal created transaction: %s", err)
	}

	txmsg, err := tx.GetMsg()
	if err != nil {
		t.Fatalf("cannot get transaction message: %s", err)
	}
	msg := txmsg.(*escrow.ReleaseEscrowMsg)

	assert.Equal(t, fromHex(t, "E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0"), []byte(msg.EscrowId))
	assert.Equal(t, []*coin.Coin{coin.NewCoinp(49, 0, "DOGE")}, msg.Amount)
}
