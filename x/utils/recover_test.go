package utils

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/stretchr/testify/assert"
)

func TestRecovery(t *testing.T) {
	var h panicHandler
	r := NewRecovery()

	ctx := context.Background()
	store := store.MemStore()

	// Panic handler panics. Test the test tool.
	assert.Panics(t, func() { h.Check(ctx, info, store, nil) })
	assert.Panics(t, func() { h.Deliver(ctx, info, store, nil) })

	// Recovery wrapped handler returns an error.
	_, err := r.Check(ctx, info, store, nil, h)
	assert.True(t, errors.ErrPanic.Is(err))

	_, err = r.Deliver(ctx, info, store, nil, h)
	assert.True(t, errors.ErrPanic.Is(err))
}

type panicHandler struct{}

var _ weave.Handler = panicHandler{}

func (p panicHandler) Check(ctx context.Context, info weave.BlockInfo, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	panic("check panic")
}

func (p panicHandler) Deliver(ctx context.Context, info weave.BlockInfo, store weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	panic("deliver panic")
}
