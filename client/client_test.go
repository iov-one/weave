package client

import (
	"context"
	"testing"

	"github.com/iov-one/weave/weavetest/assert"
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
