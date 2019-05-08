package aswap_test

import (
	"context"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/aswap"
)

func TestIsExpired(t *testing.T) {
	now := weave.AsUnixTime(time.Now())
	ctx := weave.WithBlockTime(context.Background(), now.Time())

	future := now.Add(5 * time.Minute)
	if aswap.IsExpired(ctx, future) {
		t.Error("future is expired")
	}

	past := now.Add(-5 * time.Minute)
	if !aswap.IsExpired(ctx, past) {
		t.Error("past is not expired")
	}

	if !aswap.IsExpired(ctx, now) {
		t.Fatal("when expiration time is equal to now it is expected to be expired")
	}
}

func TestIsExpiredRequiresBlockTime(t *testing.T) {
	now := weave.AsUnixTime(time.Now())
	assert.Panics(t, func() {
		// Calling aswap.IsExpected with a context without a block height
		// attached is expected to panic.
		aswap.IsExpired(context.Background(), now)
	})
}
