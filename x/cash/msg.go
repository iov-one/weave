package cash

import (
	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &SendMsg{}, migration.NoModification)
}

// Ensure we implement the Msg interface
var _ weave.Msg = (*SendMsg)(nil)

const (
	pathSendMsg                      = "cash/send"
	pathConfigurationUpdateMsg       = "cash/update_config"
	sendTxCost                 int64 = 100

	maxMemoSize int = 128
	maxRefSize  int = 64
)

// Path returns the routing path for this message
func (SendMsg) Path() string {
	return pathSendMsg
}

// Validate makes sure that this is sensible
func (s *SendMsg) Validate() error {
	var err error
	if coin.IsEmpty(s.Amount) || !s.Amount.IsPositive() {
		err = errors.Wrapf(errors.ErrAmount, "non-positive SendMsg: %#v", s.Amount)
	} else {
		err = errors.Append(err, s.Amount.Validate())
	}
	err = errors.Append(err, s.Src.Validate())
	err = errors.Append(err, s.Dest.Validate())
	if len(s.Memo) > maxMemoSize {
		err = errors.Append(err, errors.Wrap(errors.ErrState, "memo too long"))
	}
	if len(s.Ref) > maxRefSize {
		err = errors.Append(err, errors.Wrap(errors.ErrState, "ref too long"))
	}

	return err
}

// DefaultSource makes sure there is a payer.
// If it was already set, returns s.
// If none was set, returns a new SendMsg with the source set
func (s *SendMsg) DefaultSource(addr []byte) *SendMsg {
	if len(s.GetSrc()) != 0 {
		return s
	}
	return &SendMsg{
		Src:    addr,
		Dest:   s.GetDest(),
		Amount: s.GetAmount(),
		Memo:   s.GetMemo(),
		Ref:    s.GetRef(),
	}
}

// FeeTx exposes information about the fees that
// should be paid
type FeeTx interface {
	GetFees() *FeeInfo
}

// DefaultPayer makes sure there is a payer.
// If it was already set, returns f.
// If none was set, returns a new FeeInfo, with the
// New address set
func (f *FeeInfo) DefaultPayer(addr []byte) *FeeInfo {
	if len(f.GetPayer()) != 0 {
		return f
	}
	return &FeeInfo{
		Payer: addr,
		Fees:  f.GetFees(),
	}
}

// Validate makes sure that this is sensible.
// Note that fee must be present, even if 0
func (f *FeeInfo) Validate() error {
	var err error
	if f == nil {
		err = errors.Wrap(errors.ErrInput, "nil fee info")
	}
	fee := f.GetFees()
	if fee == nil {
		err = errors.Append(err, errors.Wrap(errors.ErrAmount, "fees nil"))
	} else {
		err = errors.Append(err, fee.Validate())

		if !fee.IsNonNegative() {
			err = errors.Append(err, errors.Wrap(errors.ErrAmount, "negative fees"))
		}
	}

	return errors.Append(err, weave.Address(f.Payer).Validate())
}

var _ weave.Msg = (*ConfigurationMsg)(nil)

// Validate will skip any zero fields and validate the set ones
// TODO: we should make it easier to reuse code with Configuration
func (m *ConfigurationMsg) Validate() error {
	var err error
	c := m.Patch
	if len(c.Owner) != 0 {
		err = c.Owner.Validate()
	}
	if len(c.CollectorAddress) != 0 {
		err = errors.Append(err, c.CollectorAddress.Validate())
	}
	if !c.MinimalFee.IsZero() {
		err = errors.Append(err, c.MinimalFee.Validate())

		if !c.MinimalFee.IsNonNegative() {
			err = errors.Append(err, errors.Wrap(errors.ErrState, "minimal fee cannot be negative"))
		}
	}
	return err
}

func (*ConfigurationMsg) Path() string {
	return pathConfigurationUpdateMsg
}
