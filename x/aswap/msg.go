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
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if err := validatePreimageHash(m.PreimageHash); err != nil {
		return err
	}
	if err := m.Source.Validate(); err != nil {
		return errors.Wrap(err, "src")
	}
	if err := m.Destination.Validate(); err != nil {
		return errors.Wrap(err, "destination")
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

var _ weave.Msg = (*ReleaseMsg)(nil)

func (ReleaseMsg) Path() string {
	return "aswap/release"
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

var _ weave.Msg = (*ReturnMsg)(nil)

func (ReturnMsg) Path() string {
	return "aswap/return"

}

func (m *ReturnMsg) Validate() error {
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
