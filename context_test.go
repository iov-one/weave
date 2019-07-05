package weave_test

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/tendermint/tendermint/libs/log"
)

func TestContext(t *testing.T) {
	bg := context.Background()

	// try logger with default
	newLogger := log.NewTMLogger(os.Stdout)
	ctx := weave.WithLogger(bg, newLogger)
	assert.Equal(t, weave.DefaultLogger, weave.GetLogger(bg))
	assert.Equal(t, newLogger, weave.GetLogger(ctx))

	// test height - uninitialized
	val, ok := weave.GetHeight(ctx)
	assert.Equal(t, int64(0), val)
	assert.Equal(t, false, ok)
	// set
	ctx = weave.WithHeight(ctx, 7)
	val, ok = weave.GetHeight(ctx)
	assert.Equal(t, int64(7), val)
	assert.Equal(t, true, ok)
	// no reset
	assert.Panics(t, func() { weave.WithHeight(ctx, 9) })

	// changing the info, should modify the logger, but not the height
	ctx2 := weave.WithLogInfo(ctx, "foo", "bar")
	assert.Equal(t, false, reflect.DeepEqual(weave.GetLogger(ctx2), weave.GetLogger(ctx)))
	val, _ = weave.GetHeight(ctx)
	assert.Equal(t, int64(7), val)

	// chain id MUST be set exactly once
	assert.Panics(t, func() { weave.GetChainID(ctx) })
	ctx2 = weave.WithChainID(ctx, "my-chain")
	assert.Equal(t, "my-chain", weave.GetChainID(ctx2))
	// don't try a second time
	assert.Panics(t, func() { weave.WithChainID(ctx2, "my-chain") })

	// TODO: test header context!
}

func TestChainID(t *testing.T) {
	cases := []struct {
		chainID string
		valid   bool
	}{
		{"", false},
		{"foo", false},
		{"special", true},
		{"wish-YOU-88", true},
		{"invalid;;chars", false},
		{"this-chain-id-is-way-too-long", false},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.valid, weave.IsValidChainID(tc.chainID))
	}
}
