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

func TestHeader(t *testing.T) {
	c := NewClient(NewLocalConnection(node))
	ctx := context.Background()
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
	back := context.Background()
	ctx, cancel := context.WithCancel(back)

	status, err := c.Status(ctx)
	assert.Nil(t, err)
	lastHeight := status.Height

	headers := make(chan Header, 5)
	err = c.SubscribeHeaders(ctx, headers)
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
