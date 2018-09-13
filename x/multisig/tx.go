package multisig

import (
	"github.com/iov-one/weave"
)

// MultiSigTx is an optional interface for a Tx that allows
// it to support multisig contract
type MultiSigTx interface {
	GetMultisigID() []byte
}

// MultiSigCondition returns condition for a contract ID
func MultiSigCondition(id []byte) weave.Condition {
	return weave.NewCondition("multisig", "usage", id)
}
