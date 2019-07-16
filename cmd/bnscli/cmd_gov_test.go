package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/gov"
)

func TestCmdAsProposalHappyPath(t *testing.T) {
	// Prepare a transaction that will be used as an input for the proposal
	// creation function.
	sendTx := &bnsd.Tx{
		Sum: &bnsd.Tx_CashSendMsg{
			CashSendMsg: &cash.SendMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				Source:      fromHex(t, "b1ca7e78f74423ae01da3b51e676934d9105f282"),
				Destination: fromHex(t, "E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0"),
				Amount:      coin.NewCoinp(5, 0, "DOGE"),
				Memo:        "a memo",
			},
		},
	}
	var input bytes.Buffer
	if _, err := writeTx(&input, sendTx); err != nil {
		t.Fatalf("cannot serialize transaction: %s", err)
	}

	var output bytes.Buffer
	args := []string{
		"-title", "a title",
		"-description", "a description",
		"-electionrule", "1",
	}
	if err := cmdAsProposal(&input, &output, args); err != nil {
		t.Fatalf("cannot create a new proposal transaction: %s", err)
	}

	tx, _, err := readTx(&output)
	if err != nil {
		t.Fatalf("cannot read created transaction: %s", err)
	}

	txmsg, err := tx.GetMsg()
	if err != nil {
		t.Fatalf("cannot get transaction message: %s", err)
	}
	msg := txmsg.(*gov.CreateProposalMsg)

	assert.Equal(t, msg.Title, "a title")
	assert.Equal(t, msg.Description, "a description")
	assert.Equal(t, msg.ElectionRuleID, sequenceID(1))

	var options bnsd.ProposalOptions
	if err := options.Unmarshal(msg.RawOption); err != nil {
		t.Fatalf("cannot unmarshal submessage: %s", err)
	}
	submsg := options.GetCashSendMsg()
	assert.Equal(t, fromHex(t, "b1ca7e78f74423ae01da3b51e676934d9105f282"), []byte(submsg.Source))
	assert.Equal(t, fromHex(t, "E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0"), []byte(submsg.Destination))
	assert.Equal(t, "a memo", submsg.Memo)
	assert.Equal(t, coin.NewCoinp(5, 0, "DOGE"), submsg.Amount)
}

func TestCmdAsProposalWithTextResolution(t *testing.T) {
	payloadMsg := &gov.CreateTextResolutionMsg{
		Metadata:   &weave.Metadata{Schema: 1},
		Resolution: "myTestResolution",
	}
	data, err := payloadMsg.Marshal()
	input := bytes.NewReader(data)
	var output bytes.Buffer
	args := []string{
		"-title", "a title",
		"-description", "a description",
		"-electionrule", "1",
	}
	if err := cmdAsProposal(input, &output, args); err != nil {
		t.Fatalf("cannot create a new proposal transaction: %s", err)
	}

	tx, _, err := readTx(&output)
	if err != nil {
		t.Fatalf("cannot read created transaction: %s", err)
	}

	txmsg, err := tx.GetMsg()
	if err != nil {
		t.Fatalf("cannot get transaction message: %s", err)
	}
	msg := txmsg.(*gov.CreateProposalMsg)

	assert.Equal(t, msg.Title, "a title")
	assert.Equal(t, msg.Description, "a description")
	assert.Equal(t, msg.ElectionRuleID, sequenceID(1))

	var options bnsd.ProposalOptions
	if err := options.Unmarshal(msg.RawOption); err != nil {
		t.Fatalf("cannot unmarshal submessage: %s", err)
	}
	submsg := options.GetGovCreateTextResolutionMsg()
	assert.Equal(t, "myTestResolution", submsg.Resolution)
}

func TestCmdDeleteProposalHappyPath(t *testing.T) {
	var output bytes.Buffer
	args := []string{
		"-proposal-id", "5",
	}
	if err := cmdDelProposal(nil, &output, args); err != nil {
		t.Fatalf("cannot create a new delete proposal transaction: %s", err)
	}

	tx, _, err := readTx(&output)
	if err != nil {
		t.Fatalf("cannot read created transaction: %s", err)
	}

	txmsg, err := tx.GetMsg()
	if err != nil {
		t.Fatalf("cannot get transaction message: %s", err)
	}
	msg := txmsg.(*gov.DeleteProposalMsg)

	assert.Equal(t, sequenceID(5), msg.ProposalID)
}

func TestCmdVoteHappyPath(t *testing.T) {
	var output bytes.Buffer
	args := []string{
		"-proposal-id", "5",
		"-voter", "b1ca7e78f74423ae01da3b51e676934d9105f282",
		"-select", "yes",
	}
	if err := cmdVote(nil, &output, args); err != nil {
		t.Fatalf("cannot create a new vote transaction: %s", err)
	}

	tx, _, err := readTx(&output)
	if err != nil {
		t.Fatalf("cannot read created transaction: %s", err)
	}

	txmsg, err := tx.GetMsg()
	if err != nil {
		t.Fatalf("cannot get transaction message: %s", err)
	}
	msg := txmsg.(*gov.VoteMsg)

	assert.Equal(t, sequenceID(5), msg.ProposalID)
	assert.Equal(t, fromHex(t, "b1ca7e78f74423ae01da3b51e676934d9105f282"), []byte(msg.Voter))
	assert.Equal(t, gov.VoteOption_Yes, msg.Selected)
}

func TestCmdTextResolutionHappyPath(t *testing.T) {
	var output bytes.Buffer
	args := []string{
		"-text", "myTextResolution",
	}
	if err := cmdTextResolution(nil, &output, args); err != nil {
		t.Fatalf("cannot create a text resolution msg: %s", err)
	}

	txmsg, err := readProposalPayloadMsg(&output)
	if err != nil {
		t.Fatalf("cannot get message: %s", err)
	}
	msg := txmsg.(*gov.CreateTextResolutionMsg)

	assert.Equal(t, "myTextResolution", msg.Resolution)
}

func TestCmdUpdateElectorateHappyPath(t *testing.T) {
	var output bytes.Buffer
	args := []string{
		"-id", "5",
	}
	if err := cmdUpdateElectorate(nil, &output, args); err != nil {
		t.Fatalf("cannot create a transaction: %s", err)
	}

	tx, _, err := readTx(&output)
	if err != nil {
		t.Fatalf("cannot read created transaction: %s", err)
	}

	txmsg, err := tx.GetMsg()
	if err != nil {
		t.Fatalf("cannot get transaction message: %s", err)
	}
	msg := txmsg.(*gov.UpdateElectorateMsg)

	assert.Equal(t, weavetest.SequenceID(5), msg.ElectorateID)
	assert.Equal(t, 0, len(msg.DiffElectors))
}

func TestCmdWithElectorHappyPath(t *testing.T) {
	var buf bytes.Buffer
	govTx := &bnsd.Tx{
		Sum: &bnsd.Tx_GovUpdateElectorateMsg{
			GovUpdateElectorateMsg: &gov.UpdateElectorateMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: []byte("any"),
			},
		},
	}
	_, err := writeTx(&buf, govTx)
	assert.Nil(t, err)
	var output bytes.Buffer
	args := []string{
		"-address", "b1ca7e78f74423ae01da3b51e676934d9105f282",
		"-weight", "11",
	}

	if err := cmdWithElector(&buf, &output, args); err != nil {
		t.Fatalf("cannot create a transaction: %s", err)
	}

	tx, _, err := readTx(&output)
	if err != nil {
		t.Fatalf("cannot read created transaction: %s", err)
	}

	txmsg, err := tx.GetMsg()
	if err != nil {
		t.Fatalf("cannot get transaction message: %s", err)
	}
	msg := txmsg.(*gov.UpdateElectorateMsg)

	assert.Equal(t, 1, len(msg.DiffElectors))
	expAddress := weave.Address(fromHex(t, "b1ca7e78f74423ae01da3b51e676934d9105f282"))
	assert.Equal(t, expAddress, msg.DiffElectors[0].Address)
	assert.Equal(t, uint32(11), msg.DiffElectors[0].Weight)
}

func TestCmdUpdateElectionRuleHappyPath(t *testing.T) {
	var output bytes.Buffer
	args := []string{
		"-id", "5",
		"-voting-period", "86400",
		"-threshold-numerator", "2",
		"-threshold-denominator", "3",
	}
	if err := cmdUpdateElectionRule(nil, &output, args); err != nil {
		t.Fatalf("cannot create a transaction: %s", err)
	}

	tx, _, err := readTx(&output)
	if err != nil {
		t.Fatalf("cannot read created transaction: %s", err)
	}

	txmsg, err := tx.GetMsg()
	if err != nil {
		t.Fatalf("cannot get transaction message: %s", err)
	}
	msg := txmsg.(*gov.UpdateElectionRuleMsg)

	assert.Equal(t, weavetest.SequenceID(5), msg.ElectionRuleID)
	assert.Equal(t, 24*time.Hour, msg.VotingPeriod.Duration())
	assert.Equal(t, uint32(2), msg.Threshold.Numerator)
	assert.Equal(t, uint32(3), msg.Threshold.Denominator)
}
