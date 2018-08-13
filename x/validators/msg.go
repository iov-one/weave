package validators

import (
	"github.com/iov-one/weave"
	abci "github.com/tendermint/abci/types"
)

// Ensure we implement the Msg interface
var _ weave.Msg = (*SetValidators)(nil)

const pathUpdate = "validators/update"

// Path returns the routing path for this message
func (*SetValidators) Path() string {
	return pathUpdate
}

func (m Validator) AsABCI() abci.Validator {
	return abci.Validator{
		Address: m.Address,
		PubKey:  m.PubKey.AsABCI(),
		Power:   m.Power,
	}
}

func (m PubKey) AsABCI() abci.PubKey {
	return abci.PubKey{
		Data: m.Data,
		Type: m.Type,
	}
}

func (m *SetValidators) AsABCI() []abci.Validator {
	validators := make([]abci.Validator, len(m.Validators))
	for k, v := range m.Validators {
		validators[k] = v.AsABCI()
	}

	return validators
}
