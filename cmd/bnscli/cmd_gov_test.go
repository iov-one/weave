package main

import (
	"bytes"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/gov"
)

func TestCmdAsProposalHappyPath(t *testing.T) {
	// Prepare a transaction that will be used as an input for the proposal
	// creation function.
	sendTx := &app.Tx{
		Sum: &app.Tx_SendMsg{
			SendMsg: &cash.SendMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Src:      fromHex(t, "b1ca7e78f74423ae01da3b51e676934d9105f282"),
				Dest:     fromHex(t, "E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0"),
				Amount:   coin.NewCoinp(5, 0, "DOGE"),
				Memo:     "a memo",
			},
		},
	}
	rawTx, err := sendTx.Marshal()
	if err != nil {
		t.Fatalf("cannot serialize transaction: %s", err)
	}

	var output bytes.Buffer
	args := []string{
		"-title", "a title",
		"-description", "a description",
		"-electionrule", "1",
	}
	if err := cmdAsProposal(bytes.NewReader(rawTx), &output, args); err != nil {
		t.Fatalf("cannot create a new proposal transaction: %s", err)
	}

	var tx app.Tx
	if err := tx.Unmarshal(output.Bytes()); err != nil {
		t.Fatalf("cannot unmarshal created transaction: %s", err)
	}

	txmsg, err := tx.GetMsg()
	if err != nil {
		t.Fatalf("cannot get transaction message: %s", err)
	}
	msg := txmsg.(*gov.CreateProposalMsg)

	assert.Equal(t, msg.Base.Title, "a title")
	assert.Equal(t, msg.Base.Description, "a description")
	assert.Equal(t, msg.Base.ElectionRuleID, sequenceID(1))

	var options app.ProposalOptions
	if err := options.Unmarshal(msg.RawOption); err != nil {
		t.Fatalf("cannot unmarshal submessage: %s", err)
	}
	submsg := options.GetSendMsg()
	assert.Equal(t, fromHex(t, "b1ca7e78f74423ae01da3b51e676934d9105f282"), []byte(submsg.Src))
	assert.Equal(t, fromHex(t, "E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0"), []byte(submsg.Dest))
	assert.Equal(t, "a memo", submsg.Memo)
	assert.Equal(t, coin.NewCoinp(5, 0, "DOGE"), submsg.Amount)
}
