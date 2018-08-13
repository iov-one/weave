package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/utils"
)

func TestChain(t *testing.T) {
	var help x.TestHelpers
	c1 := help.CountingDecorator()
	c2 := help.CountingDecorator()
	c3 := help.CountingDecorator()
	h := help.CountingHandler()

	stack := ChainDecorators(
		c1,
		utils.NewLogging(),
		utils.NewRecovery(),
		c2,
		help.PanicAtHeightDecorator(6),
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
	assert.Equal(t, 4, c1.GetCount())
	assert.Equal(t, 4, c2.GetCount())
	assert.Equal(t, 4, c3.GetCount())
	assert.Equal(t, 2, h.GetCount())

	// now, let's trigger a panic
	ctx = weave.WithHeight(bg, 8)
	_, err = stack.Check(ctx, nil, nil)
	assert.Error(t, err)
	_, err = stack.Deliver(ctx, nil, nil)
	assert.Error(t, err)

	assert.Equal(t, 8, c1.GetCount())
	// note that c2 is called twice in, but not out
	assert.Equal(t, 6, c2.GetCount())
	// and those two ins don't make it to c3 due to panic
	assert.Equal(t, 4, c3.GetCount())
	assert.Equal(t, 2, h.GetCount())
}
