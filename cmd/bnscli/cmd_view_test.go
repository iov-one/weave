package main

import (
	"bytes"
	"testing"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/gov"
)

func TestCmdTransactionViewHappyPath(t *testing.T) {
	tx := &bnsd.Tx{
		Sum: &bnsd.Tx_CashSendMsg{
			CashSendMsg: &cash.SendMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Memo:     "a memo",
				Ref:      []byte("123"),
			},
		},
	}
	var input bytes.Buffer
	if _, err := writeTx(&input, tx); err != nil {
		t.Fatalf("cannot marshal transaction: %s", err)
	}

	var output bytes.Buffer
	if err := cmdTransactionView(&input, &output, nil); err != nil {
		t.Fatalf("cannot view a transaction: %s", err)
	}

	const want = `{
	"Sum": {
		"CashSendMsg": {
			"metadata": {
				"schema": 1
			},
			"memo": "a memo",
			"ref": "MTIz"
		}
	}
}`
	got := output.String()

	if want != got {
		t.Logf("want: %s", want)
		t.Logf(" got: %s", got)
		t.Fatal("unexpected view result")
	}
}

func TestCmdTransactionViewWithTextResolution(t *testing.T) {
	payloadMsg := &gov.CreateTextResolutionMsg{
		Metadata:   &weave.Metadata{Schema: 1},
		Resolution: "myTestResolution",
	}
	data, _ := payloadMsg.Marshal()
	input := bytes.NewReader(data)

	var output bytes.Buffer
	if err := cmdTransactionView(input, &output, nil); err != nil {
		t.Fatalf("cannot view a transaction: %s", err)
	}
	got := output.String()
	const want = `{
	"metadata": {
		"schema": 1
	},
	"resolution": "myTestResolution"
}`
	if want != got {
		t.Logf("want: %s", want)
		t.Logf(" got: %s", got)
		t.Fatal("unexpected view result")
	}
}
