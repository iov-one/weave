package client

import (
	"testing"

	"github.com/iov-one/weave/weavetest/assert"
)

func TestWaitForNextBlock(t *testing.T) {
	c := NewClient(NewLocalConnection(node))
	ctx, cancel := timeoutCtx()
	defer cancel()

	status, err := c.Status(ctx)
	assert.Nil(t, err)
	lastHeight := status.Height

	header, err := c.WaitForNextBlock(ctx)
	assert.Nil(t, err)
	assert.Equal(t, lastHeight+1, header.Height)
}

func TestWaitForHeight(t *testing.T) {
	c := NewClient(NewLocalConnection(node))
	ctx, cancel := timeoutCtx()
	defer cancel()

	cases := map[string]struct {
		diff int64
	}{
		"next block":   {diff: 1},
		"old block":    {diff: -2},
		"future block": {diff: 3},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			status, err := c.Status(ctx)
			assert.Nil(t, err)
			desired := status.Height + tc.diff

			header, err := c.WaitForHeight(ctx, desired)
			assert.Nil(t, err)
			if header == nil {
				t.Fatalf("Returned nil header")
			}

			if tc.diff > 0 {
				// if it is the future, make sure we get correct header
				assert.Equal(t, true, desired >= header.Height)
			} else {
				// for the past, that we get the next header
				assert.Equal(t, true, status.Height+1 >= header.Height)
			}
		})
	}
}
