package escrow

import (
	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &CreateMsg{}, migration.NoModification)
	migration.MustRegister(1, &ReleaseMsg{}, migration.NoModification)
	migration.MustRegister(1, &ReturnMsg{}, migration.NoModification)
	migration.MustRegister(1, &UpdatePartiesMsg{}, migration.NoModification)
}

const (
	maxMemoSize int = 128
)

// NewCreateMsg is a helper to quickly build a create escrow message
func NewCreateMsg(
	source weave.Address,
	recipient weave.Address,
	arbiter weave.Address,
	amount coin.Coins,
	timeout weave.UnixTime,
	memo string,
) *CreateMsg {
	return &CreateMsg{
		Metadata:    &weave.Metadata{Schema: 1},
		Source:      source,
		Destination: recipient,
		Arbiter:     arbiter,
		Amount:      amount,
		Timeout:     timeout,
		Memo:        memo,
	}
}

var _ weave.Msg = (*CreateMsg)(nil)

func (CreateMsg) Path() string {
	return "escrow/create"
}

// Validate makes sure that this is sensible
func (m *CreateMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	errs = errors.AppendField(errs, "Arbiter", m.Arbiter.Validate())
	errs = errors.AppendField(errs, "Destination", m.Destination.Validate())
	if m.Timeout == 0 {
		// Zero timeout is a valid value that dates to 1970-01-01. We
		// know that this value is in the past and makes no sense. Most
		// likely value was not provided and a zero value remained.
		errs = errors.Append(errs, errors.Field("Timeout", errors.ErrInput, "required"))
	}
	errs = errors.AppendField(errs, "Timeout", m.Timeout.Validate())
	if len(m.Memo) > maxMemoSize {
		errs = errors.Append(errs, errors.Field("Memo", errors.ErrInput, "cannot be longer than %d", maxMemoSize))
	}
	errs = errors.AppendField(errs, "Amount", validateAmount(m.Amount))
	return errs
}

var _ weave.Msg = (*ReleaseMsg)(nil)

func (ReleaseMsg) Path() string {
	return "escrow/release"
}

// Validate makes sure that this is sensible
func (m *ReleaseMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	errs = errors.AppendField(errs, "EscrowID", validateEscrowID(m.EscrowId))
	if m.Amount != nil {
		errs = errors.AppendField(errs, "Amount", validateAmount(m.Amount))
	}
	return errs
}

var _ weave.Msg = (*ReturnMsg)(nil)

func (ReturnMsg) Path() string {
	return "escrow/return"
}

// Validate always returns true for no data
func (m *ReturnMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	errs = errors.AppendField(errs, "EscrowID", validateEscrowID(m.EscrowId))
	return errs
}

var _ weave.Msg = (*UpdatePartiesMsg)(nil)

func (UpdatePartiesMsg) Path() string {
	return "escrow/update"
}

// Validate makes sure any included items are valid permissions
// and there is at least one change
func (m *UpdatePartiesMsg) Validate() error {

	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	errs = errors.AppendField(errs, "EscrowID", validateEscrowID(m.EscrowId))

	// We allow for nil values because if the message does not have a field
	// set, we do not overwrite the model with a new value. At least one
	// field must be updated though.
	if m.Arbiter == nil && m.Source == nil && m.Destination == nil {
		errs = errors.Append(errs, errors.Wrap(errors.ErrEmpty, "all conditions"))
	}
	if m.Source != nil {
		errs = errors.AppendField(errs, "Source", m.Source.Validate())
	}
	if m.Destination != nil {
		errs = errors.AppendField(errs, "Destination", m.Destination.Validate())
	}
	if m.Arbiter != nil {
		errs = errors.AppendField(errs, "Arbiter", m.Arbiter.Validate())
	}
	return errs
}

func validateAmount(amount coin.Coins) error {
	// we enforce this is positive
	positive := amount.IsPositive()
	if !positive {
		return errors.Wrapf(errors.ErrAmount, "non-positive: %#v", &amount)
	}
	// then make sure these are properly formatted coins
	return amount.Validate()
}

func validateEscrowID(id []byte) error {
	switch n := len(id); {
	case n > 8:
		return errors.Wrap(errors.ErrInput, "too long")
	case n < 8:
		return errors.Wrap(errors.ErrInput, "too short")
	}
	return nil
}
