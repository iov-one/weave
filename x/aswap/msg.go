package aswap

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
}

const (
	maxMemoSize int = 128
	// preimage size in bytes
	preimageSize int = 32
	// preimageHash size in bytes
	preimageHashSize int = 32
)

var _ weave.Msg = (*CreateMsg)(nil)

func (CreateMsg) Path() string {
	return "aswap/create"
}

func (m *CreateMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	if len(m.PreimageHash) != preimageHashSize {
		errs = errors.Append(errs, errors.Field("PreimageHash", errors.ErrInput, "preimage hash has to be exactly %d bytes", preimageHashSize))
	}

	errs = errors.AppendField(errs, "Source", m.Source.Validate())
	errs = errors.AppendField(errs, "Destination", m.Destination.Validate())
	if m.Timeout == 0 {
		// Zero timeout is a valid value that dates to 1970-01-01. We
		// know that this value is in the past and makes no sense. Most
		// likely value was not provided and a zero value remained.
		errs = errors.Append(errs, errors.Field("Timeout", errors.ErrInput, "timeout is required"))
	}
	errs = errors.AppendField(errs, "Timeout", m.Timeout.Validate())
	if len(m.Memo) > maxMemoSize {
		errs = errors.Append(errs, errors.Field("Memo", errors.ErrInput, "memo must be not longer than %d characters", maxMemoSize))
	}
	if cs := coin.Coins(m.Amount); !cs.IsPositive() {
		errs = errors.Append(errs, errors.Field("Amount", errors.ErrAmount, "must be positive"))
	} else {
		errs = errors.AppendField(errs, "Amount", cs.Validate())
	}
	return errs
}

var _ weave.Msg = (*ReleaseMsg)(nil)

func (ReleaseMsg) Path() string {
	return "aswap/release"
}

func (m *ReleaseMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	errs = errors.AppendField(errs, "SwapID", validateSwapID(m.SwapID))
	if len(m.Preimage) != preimageSize {
		errs = errors.Append(errs, errors.Field("Preimage", errors.ErrInput, "preimage has to be exactly %d bytes", preimageSize))
	}
	return errs
}

var _ weave.Msg = (*ReturnMsg)(nil)

func (ReturnMsg) Path() string {
	return "aswap/return"

}

func (m *ReturnMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	errs = errors.AppendField(errs, "SwapID", validateSwapID(m.SwapID))
	return errs
}

func validateSwapID(id []byte) error {
	if len(id) != 8 {
		return errors.Wrap(errors.ErrInput, "swap ID must be 8 bytes long")
	}
	return nil
}
