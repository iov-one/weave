package escrow

import (
	"github.com/iov-one/weave"
)

// isExpired returns true if given time is in the past as compared to the "now"
// as declared for the block.
//
// This function panic if the block time is not provided in the context. This
// must never happen. The panic is here to prevent from broken setup to be
// processing data incorrectly.
func isExpired(ctx weave.Context, t weave.UnixTime) bool {
	blockNow, ok := weave.BlockTime(ctx)
	if !ok {
		panic("block time is not present")
	}
	return t.Time().Before(blockNow)
}
