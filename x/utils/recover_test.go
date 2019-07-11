package utils

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestRecovery(t *testing.T) {
	var h panicHandler
	r := NewRecovery()

	ctx := context.Background()
	store := store.MemStore()

	// Panic handler panics. Test the test tool.
	assert.Panics(t, func() { h.Check(ctx, store, nil) })
	assert.Panics(t, func() { h.Deliver(ctx, store, nil) })

	// Recovery wrapped handler returns an error.
	_, err := r.Check(ctx, store, nil, h)
	assert.IsErr(t, errors.ErrPanic, err)

	_, err = r.Deliver(ctx, store, nil, h)
	assert.IsErr(t, errors.ErrPanic, err)
}

type panicHandler struct{}

var _ weave.Handler = panicHandler{}

func (p panicHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	panic("check panic")
}

func (p panicHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	panic("deliver panic")
}
