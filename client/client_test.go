package client

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/weavetest/assert"
	cmn "github.com/tendermint/tendermint/libs/common"
	tmtypes "github.com/tendermint/tendermint/types"
)

func TestStatus(t *testing.T) {
	c := NewClient(NewLocalConnection(node))
	ctx := context.Background()
	status, err := c.Status(ctx)
	assert.Nil(t, err)
	assert.Equal(t, false, status.CatchingUp)
	if status.Height < 1 {
		t.Fatalf("Unexpected height from status: %d", status.Height)
	}
}

func TestHeader(t *testing.T) {
	c := NewClient(NewLocalConnection(node))
	ctx := context.Background()
	status, err := c.Status(ctx)
	assert.Nil(t, err)
	maxHeight := status.Height

	header, err := c.Header(ctx, maxHeight)
	assert.Nil(t, err)
	assert.Equal(t, maxHeight, header.Height)

	_, err = c.Header(ctx, maxHeight+20)
	if err == nil {
		t.Fatalf("Expected error for non-existent height")
	}
}

func TestSubscribeHeaders(t *testing.T) {
	c := NewClient(NewLocalConnection(node))
	back := context.Background()
	ctx, cancel := context.WithCancel(back)

	status, err := c.Status(ctx)
	assert.Nil(t, err)
	lastHeight := status.Height

	headers := make(chan Header, 5)
	err = c.SubscribeHeaders(ctx, headers)
	assert.Nil(t, err)

	// read three headers and ensure they are in order
	for i := 0; i < 3; i++ {
		h, ok := <-headers
		assert.Equal(t, true, ok)
		assert.Equal(t, lastHeight+1, h.Height)
		lastHeight++
	}

	// cancel the context and ensure the channel is closed
	cancel()
	_, ok := <-headers
	assert.Equal(t, false, ok)
}

func TestSubmitTx(t *testing.T) {
	c := NewClient(NewLocalConnection(node))
	ctx := context.Background()

	key := cmn.RandStr(10)
	tx := &KvTx{Key: key}
	mem, err := c.SubmitTx(ctx, tx)
	assert.Nil(t, err)
	assert.Nil(t, mem.Err)
	assert.Equal(t, tx.Hash(), mem.ID)

	// it shouldn't be available at first
	res, err := c.GetTxByID(ctx, mem.ID)
	if err == nil {
		t.Fatalf("No tx should exist yet")
	}

	// wait a block
	_, err = c.WaitForNextBlock(ctx)
	assert.Nil(t, err)
	c.WaitForTxIndex()

	// now it's there
	res, err = c.GetTxByID(ctx, mem.ID)
	assert.Nil(t, err)
	assert.Equal(t, mem.ID, res.ID)
	assert.Nil(t, res.Err)
}

type KvTx struct {
	Key   string
	Value string
}

var _ weave.Tx = (*KvTx)(nil)

func (t *KvTx) GetMsg() (weave.Msg, error) {
	return nil, nil
}

func (t *KvTx) Marshal() ([]byte, error) {
	if t.Value == "" {
		return []byte(t.Key), nil
	}
	return []byte(fmt.Sprintf("%s=%s", t.Key, t.Value)), nil
}

func (t *KvTx) Unmarshal(data []byte) error {
	parts := strings.Split(string(data), "=")
	if len(parts) == 2 {
		t.Key = parts[0]
		t.Value = parts[1]
	} else {
		t.Key = string(data)
		t.Value = string(data)
	}
	return nil
}

func (t *KvTx) Hash() TransactionID {
	bz, _ := t.Marshal()
	return tmtypes.Tx(bz).Hash()
}
