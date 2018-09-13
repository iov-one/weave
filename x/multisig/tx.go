package multisig

import (
	"crypto/sha256"

	"github.com/iov-one/weave"
)

// MultiSigTx is an optional interface for a Tx that allows
// it to support multisig contract
type MultiSigTx interface {
	GetMultiSig() weave.Address
}

// MultiSigCondition calculates a sha256 hash and then
func MultiSigCondition(id []byte) weave.Condition {
	h := sha256.Sum256(id)
	return weave.NewCondition("multisig", "usage", h[:])
}
