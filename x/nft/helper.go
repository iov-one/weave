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
	isValidAction = regexp.MustCompile(`^[A-Z_]{4,32}$`).MatchString
)

type Validation struct {
}

func (*Validation) IsValidAction(action string) bool {
	return isValidAction(action)
}

func (*Validation) IsValidTokenID(id []byte) bool {
	return len(id) >= minIDLength && len(id) <= maxIDLength
}

// TODO: Maybe fmt.Stringer for action is better, if we agree on using protobuf for that always
func FindActor(auth x.Authenticator, ctx weave.Context, t BaseNFT, action string) weave.Address {
	if auth.HasAddress(ctx, t.OwnerAddress()) {
		return t.OwnerAddress()
	} else {
		signers := x.GetAddresses(ctx, auth)
		for _, signer := range signers {
			if !t.Approvals().
				List().
				ForAction(action).
				ForAddress(signer).
				IsEmpty() {
				return signer
			}
		}
	}
	return nil
}
