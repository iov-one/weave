package cash

import (
	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &SendMsg{}, migration.NoModification)
	migration.MustRegister(1, &UpdateConfigurationMsg{}, migration.NoModification)
}

const (
	sendTxCost int64 = 100

	maxMemoSize int = 128
	maxRefSize  int = 64
)

var _ weave.Msg = (*SendMsg)(nil)

// Path returns the routing path for this message.
func (SendMsg) Path() string {
	return "cash/send"
}

// Validate makes sure that this is sensible.
func (s *SendMsg) Validate() error {
	var errs error

	if coin.IsEmpty(s.Amount) || !s.Amount.IsPositive() {
		errs = errors.Append(errs, errors.Field("Amount", errors.ErrAmount, "must be positive"))
	} else {
		errs = errors.AppendField(errs, "Amount", s.Amount.Validate())
	}
	errs = errors.AppendField(errs, "Source", s.Source.Validate())
	errs = errors.AppendField(errs, "Destination", s.Destination.Validate())
	if len(s.Memo) > maxMemoSize {
		errs = errors.Append(errs, errors.Field("Memo", errors.ErrState, "too long"))
	}
	if len(s.Ref) > maxRefSize {
		errs = errors.Append(errs, errors.Field("Ref", errors.ErrState, "too long"))
	}

	return errs
}

// DefaultSource makes sure there is a payer.
// If it was already set, returns s.
// If none was set, returns a new SendMsg with the source set
func (s *SendMsg) DefaultSource(addr []byte) *SendMsg {
	if len(s.GetSource()) != 0 {
		return s
	}
	return &SendMsg{
		Source:      addr,
		Destination: s.GetDestination(),
		Amount:      s.GetAmount(),
		Memo:        s.GetMemo(),
		Ref:         s.GetRef(),
	}
}

// FeeTx exposes information about the fees that should be paid.
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
	var errs error

	if f == nil {
		errs = errors.Append(errs, errors.Wrap(errors.ErrInput, "nil fee info"))
	}
	fee := f.GetFees()
	if fee == nil {
		errs = errors.Append(errs, errors.Wrap(errors.ErrAmount, "fees nil"))
	} else {
		errs = errors.AppendField(errs, "Fees", fee.Validate())

		if !fee.IsNonNegative() {
			errs = errors.Append(errs, errors.Field("Fees", errors.ErrAmount, "negative fees"))
		}
	}
	errs = errors.AppendField(errs, "Payer", f.Payer.Validate())

	return errs
}

var _ weave.Msg = (*UpdateConfigurationMsg)(nil)

// Validate will skip any zero fields and validate the set ones.
func (m *UpdateConfigurationMsg) Validate() error {
	var errs error
	c := m.Patch
	if len(c.Owner) != 0 {
		errs = errors.AppendField(errs, "Owner", c.Owner.Validate())
	}
	if len(c.CollectorAddress) != 0 {
		errs = errors.AppendField(errs, "CollectorAddress", c.CollectorAddress.Validate())
	}
	if !c.MinimalFee.IsZero() {
		errs = errors.AppendField(errs, "MinimalFee", c.MinimalFee.Validate())

		if !c.MinimalFee.IsNonNegative() {
			errs = errors.Append(errs, errors.Field("MinimalFee", errors.ErrState, "cannot be negative"))
		}
	}
	return errs
}

func (*UpdateConfigurationMsg) Path() string {
	return "cash/update_configuration"
}
