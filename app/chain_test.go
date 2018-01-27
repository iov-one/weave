package app

import (
	"context"
	"testing"

	"github.com/confio/weave"
	"github.com/stretchr/testify/assert"
)

func TestChain(t *testing.T) {
	c1 := new(countingDecorator)
	c2 := new(countingDecorator)
	c3 := new(countingDecorator)
	h := new(countingHandler)

	stack := ChainDecorators(
		c1,
		NewLogging(),
		NewRecovery(),
		c2,
		panicAtHeightDecorator{6},
		c3,
	).WithHandler(h)

	bg := context.Background()

	// make some calls, make sure it is fine
	_, err := stack.Check(bg, nil, nil)
	assert.NoError(t, err)
	ctx := weave.WithHeight(bg, 4)
	_, err = stack.Deliver(ctx, nil, nil)
	assert.NoError(t, err)

	// decorators are counted double, once in, once out
	assert.Equal(t, 4, c1.called)
	assert.Equal(t, 4, c2.called)
	assert.Equal(t, 4, c3.called)
	assert.Equal(t, 2, h.called)

	// now, let's trigger a panic
	ctx = weave.WithHeight(bg, 8)
	_, err = stack.Check(ctx, nil, nil)
	assert.Error(t, err)
	_, err = stack.Deliver(ctx, nil, nil)
	assert.Error(t, err)

	assert.Equal(t, 8, c1.called)
	// note that c2 is called twice in, but not out
	assert.Equal(t, 6, c2.called)
	// and those two ins don't make it to c3 due to panic
	assert.Equal(t, 4, c3.called)
	assert.Equal(t, 2, h.called)
}
