package main

import (
	"bytes"
	"testing"

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
