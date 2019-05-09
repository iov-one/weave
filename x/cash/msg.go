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
	if coin.IsEmpty(s.Amount) || !s.Amount.IsPositive() {
		return errors.Wrapf(errors.ErrAmount, "non-positive SendMsg: %#v", s.Amount)

	}
	if err := s.Amount.Validate(); err != nil {
		return errors.Wrap(err, "amount")
	}
	if err := s.Src.Validate(); err != nil {
		return errors.Wrap(err, "src")
	}
	if err := s.Dest.Validate(); err != nil {
		return errors.Wrap(err, "dst")
	}
	if len(s.Memo) > maxMemoSize {
		return errors.Wrap(errors.ErrState, "memo too long")
	}
	if len(s.Ref) > maxRefSize {
		return errors.Wrap(errors.ErrState, "ref too long")
	}
	return nil
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
	if f == nil {
		return errors.Wrap(errors.ErrInput, "nil fee info")
	}
	fee := f.GetFees()
	if fee == nil {
		return errors.Wrap(errors.ErrAmount, "fees nil")
	}
	if err := fee.Validate(); err != nil {
		return err
	}
	if !fee.IsNonNegative() {
		return errors.Wrap(errors.ErrAmount, "negative fees")
	}
	return weave.Address(f.Payer).Validate()
}

var _ weave.Msg = (*ConfigurationMsg)(nil)

// Validate will skip any zero fields and validate the set ones
// TODO: we should make it easier to reuse code with Configuration
func (m *ConfigurationMsg) Validate() error {
	c := m.Patch
	if len(c.Owner) != 0 {
		if err := c.Owner.Validate(); err != nil {
			return errors.Wrap(err, "owner address")
		}
	}
	if len(c.CollectorAddress) != 0 {
		if err := c.CollectorAddress.Validate(); err != nil {
			return errors.Wrap(err, "collector address")
		}
	}
	if !c.MinimalFee.IsZero() {
		if err := c.MinimalFee.Validate(); err != nil {
			return errors.Wrap(err, "minimal fee")
		}
		if !c.MinimalFee.IsNonNegative() {
			return errors.Wrap(errors.ErrState, "minimal fee cannot be negative")
		}
	}
	return nil
}

func (*ConfigurationMsg) Path() string {
	return pathConfigurationUpdateMsg
}
