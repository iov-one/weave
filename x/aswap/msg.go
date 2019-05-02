package aswap

import (
	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	// Migration needs to be registered for every message introduced in the codec.
	// This is the convention to message versioning.
	migration.MustRegister(1, &CreateSwapMsg{}, migration.NoModification)
	migration.MustRegister(1, &ReleaseSwapMsg{}, migration.NoModification)
	migration.MustRegister(1, &ReturnSwapMsg{}, migration.NoModification)
}

const (
	pathCreateSwap   = "swap/create"
	pathReleaseSwap  = "swap/release"
	pathReturnReturn = "swap/return"

	maxMemoSize int = 128
	// preimage size in bytes
	preimageSize int = 32
	// preimageHash size in bytes
	preimageHashSize int = 32
)

var _ weave.Msg = (*CreateSwapMsg)(nil)
var _ weave.Msg = (*ReleaseSwapMsg)(nil)
var _ weave.Msg = (*ReturnSwapMsg)(nil)

// ROUTING, Path method fulfills weave.Msg interface to allow routing

func (CreateSwapMsg) Path() string {
	return pathCreateSwap
}

func (ReleaseSwapMsg) Path() string {
	return pathReleaseSwap
}

func (ReturnSwapMsg) Path() string {
	return pathReturnReturn
}

// VALIDATION, Validate method makes sure basic rules are enforced upon input data and fulfills weave.Msg interface

func (m *CreateSwapMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if err := validatePreimageHash(m.PreimageHash); err != nil {
		return err
	}
	if err := m.Src.Validate(); err != nil {
		return errors.Wrap(err, "recipient")
	}
	if err := m.Recipient.Validate(); err != nil {
		return errors.Wrap(err, "recipient")
	}
	if m.Timeout == 0 {
		// Zero timeout is a valid value that dates to 1970-01-01. We
		// know that this value is in the past and makes no sense. Most
		// likely value was not provided and a zero value remained.
		return errors.Wrap(errors.ErrInvalidInput, "timeout is required")
	}
	if err := m.Timeout.Validate(); err != nil {
		return errors.Wrap(err, "invalid timeout value")
	}
	if len(m.Memo) > maxMemoSize {
		return errors.Wrapf(errors.ErrInvalidInput, "memo %s", m.Memo)
	}
	var err error
	m.Amount, err = validateAmount(m.Amount)
	return err
}

func (m *ReleaseSwapMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if len(m.Preimage) != preimageSize {
		return errors.Wrapf(errors.ErrInvalidInput, "preimage should be exactly %d byte long", preimageSize)
	}
	return nil
}

func (m *ReturnSwapMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	return validatePreimageHash(m.PreimageHash)
}

// validateAmount makes sure the amount is positive and coins are of valid format
func validateAmount(amount coin.Coins) (coin.Coins, error) {
	c, err := coin.NormalizeCoins(amount)
	if err != nil {
		return c, errors.Wrap(err, "unable to normalize")
	}

	positive := amount.IsPositive()
	if !positive {
		return c, errors.Wrapf(errors.ErrInvalidAmount, "non-positive CreateSwapMsg: %#v", &amount)
	}
	return c, amount.Validate()
}

func validatePreimageHash(preimageHash []byte) error {
	if len(preimageHash) != preimageHashSize {
		return errors.Wrapf(errors.ErrInvalidInput, "preimge hash is sha256 and therefore should be exactly "+
			"%d bytes", preimageHashSize)
	}
	return nil
}
