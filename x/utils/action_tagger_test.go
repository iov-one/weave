package utils

import (
	"context"
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestActionTagger(t *testing.T) {
	h := &weavetest.Handler{}
	msg := &weavetest.Msg{RoutePath: "foobar/create"}
	tx := &weavetest.Tx{Msg: msg}

	a := NewActionTagger()

	ctx := context.Background()
	store := store.MemStore()

	// ensure handler works as expected
	res, err := h.Deliver(ctx, store, tx)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(res.Tags))

	// we get tagged on success
	res, err = a.Deliver(ctx, store, tx, h)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(res.Tags))
	assert.Equal(t, "action", string(res.Tags[0].Key))
	assert.Equal(t, "foobar/create", string(res.Tags[0].Value))

	badh := &weavetest.Handler{DeliverErr: errors.ErrHuman}
	res, err = a.Deliver(ctx, store, tx, badh)
	assert.Nil(t, res)
	if !errors.ErrHuman.Is(err) {
		t.Fatalf("Expected ErrHuman, got %v", err)
	}
}
