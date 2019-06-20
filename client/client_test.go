package client

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/weavetest/assert"
	cmn "github.com/tendermint/tendermint/libs/common"
	tmtypes "github.com/tendermint/tendermint/types"
)

// defaultTimeout avoids deadlocks
var defaultTimeout = 1 * time.Second

func timeoutCtx() (context.Context, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	cancelWithWait := func() {
		cancel()
		// this makes sure the unsubscribe is finished before next test runs
		// in order to avoid multiple identical subscriptions (eg. on header) that leads to error
		time.Sleep(50 * time.Millisecond)
	}
	return ctx, cancelWithWait
}

func TestStatus(t *testing.T) {
	c := NewClient(NewLocalConnection(node))
	ctx, cancel := timeoutCtx()
	defer cancel()

	status, err := c.Status(ctx)
	assert.Nil(t, err)
	assert.Equal(t, false, status.CatchingUp)
	if status.Height < 1 {
		t.Fatalf("Unexpected height from status: %d", status.Height)
	}
}

func TestHeader(t *testing.T) {
	c := NewClient(NewLocalConnection(node))
	ctx, cancel := timeoutCtx()
	defer cancel()

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
	ctx, cancel := timeoutCtx()
	defer cancel()

	status, err := c.Status(ctx)
	assert.Nil(t, err)
	lastHeight := status.Height

	headers := make(chan Header, 5)
	err = c.SubscribeHeaders(ctx, headers, OptionCapacity{2})
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
	ctx, cancel := timeoutCtx()
	defer cancel()

	key := cmn.RandStr(10)
	tx := &KvTx{Key: key}
	id, err := c.SubmitTx(ctx, tx)
	assert.Nil(t, err)
	assert.Equal(t, tx.Hash(), id)

	// it shouldn't be available at first
	_, err = c.GetTxByID(ctx, id)
	if err == nil {
		t.Fatalf("No tx should exist yet")
	}

	// wait 2 blocks, seems flaky on the ci with one
	for i := 0; i < 2; i++ {
		_, err = c.WaitForNextBlock(ctx)
		assert.Nil(t, err)
	}

	// now it's there
	res, err := c.GetTxByID(ctx, id)
	assert.Nil(t, err)
	assert.Equal(t, id, res.ID)
	assert.Nil(t, res.Err)
}

func TestCommitTx(t *testing.T) {
	c := NewClient(NewLocalConnection(node))
	ctx, cancel := timeoutCtx()
	defer cancel()

	key := cmn.RandStr(10)
	value := cmn.RandStr(10)
	tx := &KvTx{Key: key, Value: value}
	res, err := c.CommitTx(ctx, tx)
	assert.Nil(t, err)
	assert.Nil(t, res.Err)
	assert.Equal(t, tx.Hash(), res.ID)

	tags := res.Result.Tags
	assert.Equal(t, 2, len(tags))
	assert.Equal(t, "app.key", string(tags[1].GetKey()))
	assert.Equal(t, key, string(tags[1].GetValue()))
}

func TestCommitTxs(t *testing.T) {
	c := NewClient(NewLocalConnection(node))
	ctx, cancel := timeoutCtx()
	defer cancel()

	txs := []weave.Tx{
		&KvTx{Key: cmn.RandStr(10), Value: cmn.RandStr(10)},
		&KvTx{Key: cmn.RandStr(10), Value: cmn.RandStr(10)},
		&KvTx{Key: cmn.RandStr(10), Value: cmn.RandStr(10)},
	}
	res, err := c.CommitTxs(ctx, txs)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(res))

	for i, tx := range txs {
		assert.Nil(t, res[i].Err)
		kv := (tx).(*KvTx)
		assert.Equal(t, kv.Hash(), res[i].ID)
		tags := res[i].Result.Tags
		assert.Equal(t, 2, len(tags))
		assert.Equal(t, "app.key", string(tags[1].GetKey()))
		assert.Equal(t, kv.Key, string(tags[1].GetValue()))
	}
}

func TestSearchSubscribeTx(t *testing.T) {
	c := NewClient(NewLocalConnection(node))
	ctx, cancel := timeoutCtx()
	defer cancel()

	key := cmn.RandStr(10)
	value := cmn.RandStr(10)
	tx := &KvTx{Key: key, Value: value}

	// start subscription before the commit
	query := fmt.Sprintf("app.key='%s'", key)
	sub := make(chan CommitResult, 2)
	err := c.SubscribeTx(ctx, query, sub)
	assert.Nil(t, err)

	// wait until it is in the block
	res, err := c.CommitTx(ctx, tx)
	assert.Nil(t, err)
	assert.Nil(t, res.Err)
	assert.Equal(t, res.Result.Tags[1].Key, []byte("app.key"))
	assert.Equal(t, res.Result.Tags[1].Value, []byte(key))

	// now search for a transaction
	matches, err := c.SearchTx(ctx, query)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(matches))
	fromSearch := matches[0]
	assert.Nil(t, fromSearch.Err)
	assert.Equal(t, tx.Hash(), fromSearch.ID)

	// and make sure the same was received in subscription
	fromSub, ok := <-sub
	assert.Equal(t, true, ok)
	assert.Nil(t, fromSub.Err)
	assert.Equal(t, tx.Hash(), fromSub.ID)

	// are they both identical
	assert.Equal(t, fromSub, *fromSearch)
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
