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

func FindActor(height int64, auth x.Authenticator, ctx weave.Context, t BaseNFT, action string) weave.Address {
	if auth.HasAddress(ctx, t.OwnerAddress()) {
		return t.OwnerAddress()
	}

	signers := x.GetAddresses(ctx, auth)
	for _, signer := range signers {
		if !t.Approvals().
			List().
			ForAction(action).
			ForAddress(signer).
			FilterExpired(height).
			IsEmpty() {
			return signer
		}
	}
	return nil
}
