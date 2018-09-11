package multisig

import (
	"github.com/iov-one/weave"
)

// MultiSigTx is an optional interface for a Tx that allows
// it to support multisig contract
type MultiSigTx interface {
	GetMultiSig() weave.Address
}

func MultiSigConditions(contract Contract) []weave.Condition {
	var conditions []weave.Condition
	for _, sig := range contract.Sigs {
		conditions = append(conditions, weave.Condition(sig))
	}
	return conditions
}
