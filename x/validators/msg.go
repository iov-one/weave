package validators

import (
	fmt "fmt"

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
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	if len(m.ValidatorUpdates) == 0 {
		errs = errors.AppendField(errs, "ValidatorUpdates", errors.ErrEmpty)
	}
	for i, v := range m.ValidatorUpdates {
		errs = errors.AppendField(errs, fmt.Sprintf("ValidatorUpdates.%d", i), v.Validate())
	}
	return errs
}

func (m *ApplyDiffMsg) AsABCI() []abci.ValidatorUpdate {
	validators := make([]abci.ValidatorUpdate, len(m.ValidatorUpdates))
	for k, v := range m.ValidatorUpdates {
		validators[k] = v.AsABCI()
	}

	return validators
}
