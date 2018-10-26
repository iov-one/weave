package validators

import (
	"github.com/iov-one/weave"
	abci "github.com/tendermint/tendermint/abci/types"
)

// Ensure we implement the Msg interface
var _ weave.Msg = (*SetValidatorsMsg)(nil)

const pathUpdate = "validators/update"

// Path returns the routing path for this message
func (*SetValidatorsMsg) Path() string {
	return pathUpdate
}

func (m Validator) AsABCI() abci.Validator {
	return abci.Validator{
		Address: m.Address,
		PubKey:  m.Pubkey.AsABCI(),
		Power:   m.Power,
	}
}

func (m Pubkey) AsABCI() abci.PubKey {
	return abci.PubKey{
		Data: m.Data,
		Type: m.Type,
	}
}

func (m *SetValidatorsMsg) AsABCI() []abci.Validator {
	validators := make([]abci.Validator, len(m.Validators))
	for k, v := range m.Validators {
		validators[k] = v.AsABCI()
	}

	return validators
}
