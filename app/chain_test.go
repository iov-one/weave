package app

import (
	"context"
	"reflect"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/utils"
)

func TestChain(t *testing.T) {
	c1 := &weavetest.Decorator{}
	c2 := &weavetest.Decorator{}
	c3 := &weavetest.Decorator{}
	h := &weavetest.Handler{}

	const panicHeight = 8

	stack := ChainDecorators(
		c1,
		utils.NewLogging(),
		utils.NewRecovery(),
		c2,
		panicAtHeightDecorator(panicHeight),
		c3,
	).WithHandler(h)

	// This height must not panic.
	ctx := weave.WithHeight(context.Background(), panicHeight-2)

	_, err := stack.Check(ctx, nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, 1, c1.CheckCallCount())
	assert.Equal(t, 1, c2.CheckCallCount())
	assert.Equal(t, 1, c3.CheckCallCount())
	assert.Equal(t, 1, h.CheckCallCount())

	_, err = stack.Deliver(ctx, nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, 1, c1.DeliverCallCount())
	assert.Equal(t, 1, c2.DeliverCallCount())
	assert.Equal(t, 1, c3.DeliverCallCount())
	assert.Equal(t, 1, h.DeliverCallCount())

	// Trigger a panic.
	ctx = weave.WithHeight(context.Background(), panicHeight+2)

	_, err = stack.Check(ctx, nil, nil)
	assert.IsErr(t, errors.ErrPanic, err)
	assert.Equal(t, 2, c1.CheckCallCount())
	assert.Equal(t, 2, c2.CheckCallCount())
	assert.Equal(t, 1, c3.CheckCallCount())
	assert.Equal(t, 1, h.CheckCallCount())

	_, err = stack.Deliver(ctx, nil, nil)
	assert.IsErr(t, errors.ErrPanic, err)
	assert.Equal(t, 2, c1.DeliverCallCount())
	assert.Equal(t, 2, c2.DeliverCallCount())
	assert.Equal(t, 1, c3.DeliverCallCount())
	assert.Equal(t, 1, h.DeliverCallCount())
}

type panicAtHeightDecorator int64

var _ weave.Decorator = panicAtHeightDecorator(0)

func (ph panicAtHeightDecorator) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx, next weave.Checker) (*weave.CheckResult, error) {
	if val, _ := weave.GetHeight(ctx); val >= int64(ph) {
		panic("too high")
	}
	return next.Check(ctx, db, tx)
}

func (ph panicAtHeightDecorator) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx, next weave.Deliverer) (*weave.DeliverResult, error) {
	if val, _ := weave.GetHeight(ctx); val >= int64(ph) {
		panic("too high")
	}
	return next.Deliver(ctx, db, tx)
}

func TestChainNilDecorator(t *testing.T) {
	stack := ChainDecorators(nil, &weavetest.Decorator{}, nil, nil)
	if want, got := 1, len(stack.chain); want != got {
		t.Fatalf("want %d decorator, got %d", want, got)
	}

	stack = stack.Chain(nil, &weavetest.Decorator{}, nil, nil)
	if want, got := 2, len(stack.chain); want != got {
		t.Fatalf("want %d decorators, got %d", want, got)
	}
}

func TestCutoffNil(t *testing.T) {
	// D is a custom implementation that allows instances to can be
	// compared by the ID.
	type D struct {
		weave.Decorator
		ID int
	}

	cases := map[string]struct {
		input []weave.Decorator
		want  []weave.Decorator
	}{
		"nil input": {
			input: nil,
			want:  nil,
		},
		"empty input": {
			input: []weave.Decorator{},
			want:  []weave.Decorator{},
		},
		"only nil": {
			input: []weave.Decorator{nil, nil},
			want:  []weave.Decorator{},
		},
		"single decorator": {
			input: []weave.Decorator{&D{ID: 1}},
			want:  []weave.Decorator{&D{ID: 1}},
		},
		"order is preserved": {
			input: []weave.Decorator{
				nil, &D{ID: 1}, nil,
				nil, &D{ID: 2}, nil,
				nil, &D{ID: 3}, nil,
			},
			want: []weave.Decorator{
				&D{ID: 1}, &D{ID: 2}, &D{ID: 3},
			},
		},
		"surrounded by nil": {
			input: []weave.Decorator{nil, &D{ID: 1}, nil},
			want:  []weave.Decorator{&D{ID: 1}},
		},
		"nil on left": {
			input: []weave.Decorator{nil, nil, nil, &D{ID: 1}},
			want:  []weave.Decorator{&D{ID: 1}},
		},
		"nil on right": {
			input: []weave.Decorator{&D{ID: 1}, nil, nil, nil},
			want:  []weave.Decorator{&D{ID: 1}},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			got := cutoffNil(tc.input)
			if !reflect.DeepEqual(tc.want, got) {
				t.Fatalf("unexpected result: %s", tc.want)
			}
		})
	}
}
