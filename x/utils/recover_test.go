package utils

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
)

func TestRecovery(t *testing.T) {
	var help x.TestHelpers

	pan := help.PanicHandler(fmt.Errorf("boom"))
	r := NewRecovery()

	ctx := context.Background()
	store := store.MemStore()

	// panic handler panics
	assert.Panics(t, func() { pan.Check(ctx, store, nil) })
	assert.Panics(t, func() { pan.Deliver(ctx, store, nil) })

	// recovery wrapped handler returns error
	_, err := r.Check(ctx, store, nil, pan)
	assert.Error(t, err)
	assert.Equal(t, "panic: boom", err.Error())
	_, err = r.Deliver(ctx, store, nil, pan)
	assert.Error(t, err)
	assert.Equal(t, "panic: boom", err.Error())
}
