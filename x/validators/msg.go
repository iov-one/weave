package validators

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	abci "github.com/tendermint/tendermint/abci/types"
)

func init() {
	migration.MustRegister(1, &ApplyDiffMsg{}, migration.NoModification)
}

var _ weave.Msg = (*ApplyDiffMsg)(nil)

// Path implements weave.Msg interface.
func (*ApplyDiffMsg) Path() string {
	return "validators/apply_diff"
}

func (m *ApplyDiffMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if len(m.ValidatorUpdates) == 0 {
		return errors.Wrap(errors.ErrEmpty, "validator set")
	}
	for _, v := range m.ValidatorUpdates {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (m *ApplyDiffMsg) AsABCI() []abci.ValidatorUpdate {
	validators := make([]abci.ValidatorUpdate, len(m.ValidatorUpdates))
	for k, v := range m.ValidatorUpdates {
		validators[k] = v.AsABCI()
	}

	return validators
}
