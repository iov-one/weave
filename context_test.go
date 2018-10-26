package weave

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tendermint/tendermint/libs/log"
)

func TestContext(t *testing.T) {
	bg := context.Background()

	// try logger with default
	newLogger := log.NewTMLogger(os.Stdout)
	ctx := WithLogger(bg, newLogger)
	assert.Equal(t, DefaultLogger, GetLogger(bg))
	assert.Equal(t, newLogger, GetLogger(ctx))

	// test height - uninitialized
	val, ok := GetHeight(ctx)
	assert.Equal(t, int64(0), val)
	assert.False(t, ok)
	// set
	ctx = WithHeight(ctx, 7)
	val, ok = GetHeight(ctx)
	assert.Equal(t, int64(7), val)
	assert.True(t, ok)
	// no reset
	assert.Panics(t, func() { WithHeight(ctx, 9) })

	// changing the info, should modify the logger, but not the height
	ctx2 := WithLogInfo(ctx, "foo", "bar")
	assert.NotEqual(t, GetLogger(ctx), GetLogger(ctx2))
	val, _ = GetHeight(ctx)
	assert.Equal(t, int64(7), val)

	// chain id MUST be set exactly once
	assert.Panics(t, func() { GetChainID(ctx) })
	ctx2 = WithChainID(ctx, "my-chain")
	assert.Equal(t, "my-chain", GetChainID(ctx2))
	// don't try a second time
	assert.Panics(t, func() { WithChainID(ctx2, "my-chain") })

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
		assert.Equal(t, tc.valid, IsValidChainID(tc.chainID), tc.chainID)
	}
}
