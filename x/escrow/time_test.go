package escrow

import (
	"context"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestIsExpired(t *testing.T) {
	now := weave.AsUnixTime(time.Now())
	ctx := weave.WithBlockTime(context.Background(), now.Time())

	future := now.Add(5 * time.Minute)
	if isExpired(ctx, future) {
		t.Error("future is expired")
	}

	past := now.Add(-5 * time.Minute)
	if !isExpired(ctx, past) {
		t.Error("past is not expired")
	}
}

func TestIsExpiredRequiresBlockTime(t *testing.T) {
	now := weave.AsUnixTime(time.Now())
	assert.Panics(t, func() {
		// Calling isExpected with a context without a block height
		// attached is expected to panic.
		isExpired(context.Background(), now)
	})
}
