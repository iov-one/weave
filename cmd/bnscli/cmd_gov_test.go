package main

import (
	"bytes"
	"testing"

	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/gov"
)

func TestCmdNewTransferProposalHappyPath(t *testing.T) {
	var output bytes.Buffer
	args := []string{
		"-src", "b1ca7e78f74423ae01da3b51e676934d9105f282",
		"-dst", "E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0",
		"-amount", "5 DOGE",
		"-memo", "a memo",
		"-title", "a title",
		"-description", "a description",
		"-electionrule", "1",
	}
	if err := cmdNewTransferProposal(nil, &output, args); err != nil {
		t.Fatalf("cannot create a new transfer proposal: %s", err)
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

func TestCmdNewEscrowProposalHappyPath(t *testing.T) {
	var output bytes.Buffer
	args := []string{
		"-escrow", "E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0",
		"-amount", "49 DOGE",
		"-title", "a title",
		"-description", "a description",
		"-electionrule", "1",
	}
	if err := cmdNewEscrowProposal(nil, &output, args); err != nil {
		t.Fatalf("cannot create a new transfer proposal: %s", err)
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
	submsg := options.GetReleaseEscrowMsg()
	assert.Equal(t, fromHex(t, "E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0"), []byte(submsg.EscrowId))
	assert.Equal(t, []*coin.Coin{coin.NewCoinp(49, 0, "DOGE")}, submsg.Amount)
}
