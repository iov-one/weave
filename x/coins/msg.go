package coins

import (
	"github.com/confio/weave"
	"github.com/confio/weave/errors"
)

// Ensure we implement the Msg interface
var _ weave.Msg = (*SendMsg)(nil)

const (
	pathSendMsg       = "coins/send"
	maxNoteSize int   = 250
	sendTxCost  int64 = 100
)

// Path returns the routing path for this message
func (SendMsg) Path() string {
	return pathSendMsg
}

// Validate makes sure that this is sensible
func (s *SendMsg) Validate() error {
	if !s.GetAmount().IsPositive() {
		return ErrInvalidAmount("Non-positive SendMsg")
	}
	l := weave.AddressLength
	if len(s.GetSrc()) != l {
		return errors.ErrUnrecognizedAddress(s.GetSrc())
	}
	if len(s.GetDest()) != l {
		return errors.ErrUnrecognizedAddress(s.GetDest())
	}
	if len(s.GetNote()) > maxNoteSize {
		return ErrInvalidMemo("Note too long")
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
		Note:   s.GetNote(),
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

// Validate makes sure that this is sensible
func (f *FeeInfo) Validate() error {
	if !f.GetFees().IsNonNegative() {
		return ErrInvalidAmount("Negative fees")
	}
	l := weave.AddressLength
	if len(f.GetPayer()) != l {
		return errors.ErrUnrecognizedAddress(f.GetPayer())
	}
	return nil
}
