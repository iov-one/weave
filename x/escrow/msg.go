package escrow

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
)

const (
	pathCreateEscrowMsg        = "escrow/create"
	pathReleaseEscrowMsg       = "escrow/release"
	pathReturnEscrowMsg        = "escrow/return"
	pathUpdateEscrowPartiesMsg = "escrow/update"

	maxMemoSize int = 128
)

var _ weave.Msg = (*CreateEscrowMsg)(nil)
var _ weave.Msg = (*ReleaseEscrowMsg)(nil)
var _ weave.Msg = (*ReturnEscrowMsg)(nil)
var _ weave.Msg = (*UpdateEscrowPartiesMsg)(nil)

//--------- Path routing --------

// Path fulfills weave.Msg interface to allow routing
func (CreateEscrowMsg) Path() string {
	return pathCreateEscrowMsg
}

// Path fulfills weave.Msg interface to allow routing
func (ReleaseEscrowMsg) Path() string {
	return pathReleaseEscrowMsg
}

// Path fulfills weave.Msg interface to allow routing
func (ReturnEscrowMsg) Path() string {
	return pathReturnEscrowMsg
}

// Path fulfills weave.Msg interface to allow routing
func (UpdateEscrowPartiesMsg) Path() string {
	return pathUpdateEscrowPartiesMsg
}

//--------- Validation --------

// NewCreateMsg is a helper to quickly build a create escrow message
func NewCreateMsg(send, rcpt weave.Address, arb weave.Condition,
	amount x.Coins, timeout int64, memo string) *CreateEscrowMsg {
	return &CreateEscrowMsg{
		Sender:    send,
		Recipient: rcpt,
		Arbiter:   arb,
		Amount:    amount,
		Timeout:   timeout,
		Memo:      memo,
	}
}

// Validate makes sure that this is sensible
func (m *CreateEscrowMsg) Validate() error {
	if m.Arbiter == nil {
		return ErrMissingArbiter()
	}
	if m.Recipient == nil {
		return ErrMissingRecipient()
	}
	if m.Timeout <= 0 {
		return ErrInvalidTimeout(m.Timeout)
	}
	if len(m.Memo) > maxMemoSize {
		return ErrInvalidMemo(m.Memo)
	}
	if err := validateAmount(m.Amount); err != nil {
		return err
	}
	if err := validateConditions(m.Arbiter); err != nil {
		return err
	}
	return validateAddresses(m.Sender, m.Recipient)
}

// Validate makes sure that this is sensible
func (m *ReleaseEscrowMsg) Validate() error {
	err := validateEscrowID(m.EscrowId)
	if err != nil {
		return err
	}
	if m.Amount == nil {
		return nil
	}
	return validateAmount(m.Amount)
}

// Validate always returns true for no data
func (m *ReturnEscrowMsg) Validate() error {
	return validateEscrowID(m.EscrowId)
}

// Validate makes sure any included items are valid permissions
// and there is at least one change
func (m *UpdateEscrowPartiesMsg) Validate() error {
	err := validateEscrowID(m.EscrowId)
	if err != nil {
		return err
	}
	if m.Arbiter == nil &&
		m.Sender == nil &&
		m.Recipient == nil {
		return ErrMissingAllConditions()
	}
	err = validateConditions(m.Arbiter)
	if err != nil {
		return err
	}
	return validateAddresses(m.Sender, m.Recipient)
}

// validateConditions returns an error if any permission doesn't validate
// nil is considered valid here
func validateConditions(perms ...weave.Condition) error {
	for _, p := range perms {
		if p != nil {
			if err := p.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
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

func validateAmount(amount x.Coins) error {
	// we enforce this is positive
	positive := amount.IsPositive()
	if !positive {
		return cash.ErrInvalidAmount("Non-positive SendMsg")
	}
	// then make sure these are properly formatted coins
	return amount.Validate()
}

func validateEscrowID(id []byte) error {
	if len(id) != 8 {
		return ErrInvalidEscrowID(id)
	}
	return nil
}
