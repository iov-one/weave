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

func (m ValidatorUpdates) Validate() error {
	var err error
	for _, v := range m.ValidatorUpdates {
		err = errors.Append(err, v.Validate())
	}
	return err
}

func (m ValidatorUpdate) Validate() error {
	if len(m.Pubkey.Data) != 32 || strings.ToLower(m.Pubkey.Type) != "ed25519" {
		return errors.Wrapf(errors.ErrType, "invalid public key: %T", m.Pubkey.Type)
	}
	if m.Power < 0 {
		return errors.Wrapf(errors.ErrMsg, "power: %d", m.Power)
	}
	return nil
}

func (m ValidatorUpdate) AsABCI() abci.ValidatorUpdate {
	return abci.ValidatorUpdate{
		PubKey: m.Pubkey.AsABCI(),
		Power:  m.Power,
	}
}

func ValidatorUpdatesFromABCI(u []abci.ValidatorUpdate) ValidatorUpdates {
	vu := ValidatorUpdates{
		ValidatorUpdates: make([]ValidatorUpdate, len(u)),
	}

	for k, v := range u {
		vu.ValidatorUpdates[k] = ValidatorUpdateFromABCI(v)
	}

	return vu
}

func ValidatorUpdateFromABCI(u abci.ValidatorUpdate) ValidatorUpdate {
	return ValidatorUpdate{
		Power:  u.Power,
		Pubkey: PubkeyFromABCI(u.PubKey),
	}
}

func PubkeyFromABCI(u abci.PubKey) Pubkey {
	return Pubkey{
		Type: u.Type,
		Data: u.Data,
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
			return errors.Wrap(errors.ErrInput, "validator set must not contain nil ")
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
