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
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if err := m.Arbiter.Validate(); err != nil {
		return errors.Wrap(err, "arbiter")
	}
	if err := m.Destination.Validate(); err != nil {
		return errors.Wrap(err, "recipient")
	}
	if m.Timeout == 0 {
		// Zero timeout is a valid value that dates to 1970-01-01. We
		// know that this value is in the past and makes no sense. Most
		// likely value was not provided and a zero value remained.
		return errors.Wrap(errors.ErrInput, "timeout is required")
	}
	if err := m.Timeout.Validate(); err != nil {
		return errors.Wrap(err, "invalid timeout value")
	}
	if len(m.Memo) > maxMemoSize {
		return errors.Wrapf(errors.ErrInput, "memo %s", m.Memo)
	}
	if err := validateAmount(m.Amount); err != nil {
		return err
	}
	return nil
}

var _ weave.Msg = (*ReleaseMsg)(nil)

func (ReleaseMsg) Path() string {
	return "escrow/release"
}

// Validate makes sure that this is sensible
func (m *ReleaseMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	err := validateEscrowID(m.EscrowId)
	if err != nil {
		return err
	}
	if m.Amount == nil {
		return nil
	}
	return validateAmount(m.Amount)
}

var _ weave.Msg = (*ReturnMsg)(nil)

func (ReturnMsg) Path() string {
	return "escrow/return"
}

// Validate always returns true for no data
func (m *ReturnMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	return validateEscrowID(m.EscrowId)
}

var _ weave.Msg = (*UpdatePartiesMsg)(nil)

func (UpdatePartiesMsg) Path() string {
	return "escrow/update"
}

// Validate makes sure any included items are valid permissions
// and there is at least one change
func (m *UpdatePartiesMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	err := validateEscrowID(m.EscrowId)
	if err != nil {
		return err
	}
	if m.Arbiter == nil && m.Source == nil && m.Destination == nil {
		return errors.Wrap(errors.ErrEmpty, "all conditions")
	}

	return validateAddresses(m.Source, m.Destination, m.Arbiter)
}

// validateAddresses returns an error if any address doesn't validate
// nil is considered valid here
func validateAddresses(addrs ...weave.Address) error {
	for _, a := range addrs {
		if a != nil {
			if err := a.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateAmount(amount coin.Coins) error {
	// we enforce this is positive
	positive := amount.IsPositive()
	if !positive {
		return errors.Wrapf(errors.ErrAmount, "non-positive SendMsg: %#v", &amount)
	}
	// then make sure these are properly formatted coins
	return amount.Validate()
}

func validateEscrowID(id []byte) error {
	if len(id) != 8 {
		return errors.Wrapf(errors.ErrInput, "escrow id: %X", id)
	}
	return nil
}
