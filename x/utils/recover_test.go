package utils

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/stretchr/testify/assert"
)

//nolint
func TestRecovery(t *testing.T) {
	var h panicHandler
	r := NewRecovery()

	ctx := context.Background()
	s := store.MemStore()

	// Panic handler panics. Test the test tool.
	assert.Panics(t, func() { h.Check(ctx, s, nil) })
	assert.Panics(t, func() { h.Deliver(ctx, s, nil) })

	// Recovery wrapped handler returns an error.
	_, err := r.Check(ctx, s, nil, h)
	assert.True(t, errors.ErrPanic.Is(err))

	_, err = r.Deliver(ctx, s, nil, h)
	assert.True(t, errors.ErrPanic.Is(err))
}

type panicHandler struct{}

var _ weave.Handler = panicHandler{}

func (p panicHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	panic("check panic")
}

func (p panicHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	panic("deliver panic")
}
