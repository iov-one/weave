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
	migration.MustRegister(1, &CreateMsg{}, migration.NoModification)
	migration.MustRegister(1, &ReleaseMsg{}, migration.NoModification)
	migration.MustRegister(1, &ReturnSwapMsg{}, migration.NoModification)
}

const (
	pathCreateSwap  = "aswap/create"
	pathReleaseSwap = "aswap/release"
	pathReturnSwap  = "aswap/return"

	maxMemoSize int = 128
	// preimage size in bytes
	preimageSize int = 32
	// preimageHash size in bytes
	preimageHashSize int = 32
)

var _ weave.Msg = (*CreateMsg)(nil)
var _ weave.Msg = (*ReleaseMsg)(nil)
var _ weave.Msg = (*ReturnSwapMsg)(nil)

// ROUTING, Path method fulfills weave.Msg interface to allow routing

func (CreateMsg) Path() string {
	return pathCreateSwap
}

func (ReleaseMsg) Path() string {
	return pathReleaseSwap
}

func (ReturnSwapMsg) Path() string {
	return pathReturnSwap
}

// VALIDATION, Validate method makes sure basic rules are enforced upon input data and fulfills weave.Msg interface

func (m *CreateMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if err := validatePreimageHash(m.PreimageHash); err != nil {
		return err
	}
	if err := m.Src.Validate(); err != nil {
		return errors.Wrap(err, "src")
	}
	if err := m.Recipient.Validate(); err != nil {
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
	var err error
	m.Amount, err = validateAmount(m.Amount)
	return err
}

func (m *ReleaseMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}

	if err := validateSwapID(m.SwapID); err != nil {
		return errors.Wrap(err, "SwapID")
	}

	if len(m.Preimage) != preimageSize {
		return errors.Wrapf(errors.ErrInput, "preimage should be exactly %d byte long", preimageSize)
	}
	return nil
}

func (m *ReturnSwapMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if err := validateSwapID(m.SwapID); err != nil {
		return errors.Wrap(err, "SwapID")
	}
	return nil
}

// validateAmount makes sure the amount is positive and coins are of valid format
func validateAmount(amount coin.Coins) (coin.Coins, error) {
	c, err := coin.NormalizeCoins(amount)
	if err != nil {
		return c, errors.Wrap(err, "unable to normalize")
	}

	positive := c.IsPositive()
	if !positive {
		return c, errors.Wrapf(errors.ErrAmount, "non-positive CreateMsg: %#v", &c)
	}
	return c, c.Validate()
}

func validatePreimageHash(preimageHash []byte) error {
	if len(preimageHash) != preimageHashSize {
		return errors.Wrapf(errors.ErrInput, "preimge hash is sha256 and therefore should be exactly "+
			"%d bytes", preimageHashSize)
	}
	return nil
}

func validateSwapID(id []byte) error {
	if len(id) != 8 {
		return errors.Wrapf(errors.ErrInput, "SwapID: %X", id)
	}
	return nil
}
