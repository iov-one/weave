package aswap

import (
	"github.com/iov-one/weave"
)

// IsExpired returns true if given time is in the past as compared to the "now"
// as declared for the block. Expiration is inclusive, meaning that if current
// time is equal to the expiration time than this function returns true.
//
// This function panic if the block time is not provided in the context. This
// must never happen. The panic is here to prevent from broken setup to be
// processing data incorrectly.
func IsExpired(ctx weave.Context, t weave.UnixTime) bool {
	blockNow, ok := weave.BlockTime(ctx)
	if !ok {
		panic("block time is not present")
	}
	return t <= weave.AsUnixTime(blockNow)
}
