package validators

import (
	"strings"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	abci "github.com/tendermint/tendermint/abci/types"
)

func init() {
	migration.MustRegister(1, &SetValidatorsMsg{}, migration.NoModification)
}

// Ensure we implement the Msg interface
var _ weave.Msg = (*SetValidatorsMsg)(nil)

const pathUpdate = "validators/update"

// Path returns the routing path for this message
func (*SetValidatorsMsg) Path() string {
	return pathUpdate
}

func (m ValidatorUpdate) Validate() error {
	if len(m.Pubkey.Data) != 32 ||
		strings.ToLower(m.Pubkey.Type) != "ed25519" {
		return errors.Wrap(ErrInvalidPubKey, m.Pubkey.Type)
	}
	if m.Power < 0 {
		return errors.Wrapf(ErrInvalidPower, "%d", m.Power)
	}
	return nil
}

func (m ValidatorUpdate) AsABCI() abci.ValidatorUpdate {
	return abci.ValidatorUpdate{
		PubKey: m.Pubkey.AsABCI(),
		Power:  m.Power,
	}
}

func (m Pubkey) AsABCI() abci.PubKey {
	return abci.PubKey{
		Data: m.Data,
		Type: m.Type,
	}
}

func (m *SetValidatorsMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if len(m.ValidatorUpdates) == 0 {
		return errors.Wrap(errors.ErrEmpty, "validator set")
	}
	for _, v := range m.ValidatorUpdates {
		if v == nil {
			return errors.Wrap(errors.ErrInvalidInput, "validator set must not contain nil ")
		}
		if err := v.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (m *SetValidatorsMsg) AsABCI() []abci.ValidatorUpdate {
	validators := make([]abci.ValidatorUpdate, len(m.ValidatorUpdates))
	for k, v := range m.ValidatorUpdates {
		validators[k] = v.AsABCI()
	}

	return validators
}
