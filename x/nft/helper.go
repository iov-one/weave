package nft

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
)

const (
	minIDLength = 3
	maxIDLength = 256
)

func isValidTokenID(id []byte) bool {
	return len(id) >= minIDLength && len(id) <= maxIDLength
}

func FindActor(auth x.Authenticator, ctx weave.Context, t BaseNFT, action Action) weave.Address {
	if auth.HasAddress(ctx, t.OwnerAddress()) {
		return t.OwnerAddress()
	}

	height, _ := weave.GetHeight(ctx)
	signers := x.GetAddresses(ctx, auth)
	for _, signer := range signers {
		alive := t.Approvals().List().ForAction(action).ForAddress(signer).FilterExpired(height)
		if !alive.IsEmpty() {
			t.SetApprovals(t.Approvals().List().MergeUsed(alive.UseCount()))
			return signer
		}
	}

	return nil
}
