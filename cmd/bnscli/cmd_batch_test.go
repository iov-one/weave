package main

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/batch"
	"github.com/iov-one/weave/x/cash"
)

func TestCmdAsBatchHappyPath(t *testing.T) {
	var input bytes.Buffer
	for i := 0; i < 3; i++ {
		tx := &bnsd.Tx{
			Sum: &bnsd.Tx_CashSendMsg{
				CashSendMsg: &cash.SendMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Memo:     fmt.Sprintf("memo %d", i),
				},
			},
		}
		if _, err := writeTx(&input, tx); err != nil {
			t.Fatalf("cannot marshal transaction: %s", err)
		}
	}

	var output bytes.Buffer
	if err := cmdAsBatch(&input, &output, nil); err != nil {
		t.Fatalf("cannot create a batch transaction: %s", err)
	}

	tx, _, err := readTx(&output)
	if err != nil {
		t.Fatalf("cannot read batch transaction: %s", err)
	}
	msg, err := tx.GetMsg()
	if err != nil {
		t.Fatalf("cannot get message: %s", err)
	}
	bmsg, ok := msg.(batch.Msg)
	if !ok {
		t.Fatalf("not a batch message: %T", msg)
	}
	msgs, err := bmsg.MsgList()
	if err != nil {
		t.Fatalf("cannot get messages list: %s", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}

	assert.Equal(t, "memo 0", msgs[0].(*cash.SendMsg).Memo)
	assert.Equal(t, "memo 1", msgs[1].(*cash.SendMsg).Memo)
	assert.Equal(t, "memo 2", msgs[2].(*cash.SendMsg).Memo)
}
