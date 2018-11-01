package nft

import (
	"regexp"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
)

const (
	minIDLength = 3
	maxIDLength = 256
)

var (
	isValidAction = regexp.MustCompile(`^[A-Za-z]{4,32}$`).MatchString
)

type Validation struct {
}

func (*Validation) IsValidAction(action string) bool {
	return isValidAction(action)
}

func (*Validation) IsValidTokenID(id []byte) bool {
	return len(id) >= minIDLength && len(id) <= maxIDLength
}

func FindActor(auth x.Authenticator, ctx weave.Context, t BaseNFT, action string) weave.Address {
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
