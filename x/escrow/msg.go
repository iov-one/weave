package escrow

import (
	"errors"

	"github.com/confio/weave"
	"github.com/confio/weave/x"
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

// Validate makes sure that this is sensible
func (m *CreateEscrowMsg) Validate() error {
	if m.Arbiter == nil {
		return errors.New("TODO: missing arbiter")
	}
	if m.Recipient == nil {
		return errors.New("TODO: missing recipient")
	}
	if m.Timeout <= 0 {
		return errors.New("TODO: invalid timeout")
	}
	if len(m.Memo) > maxMemoSize {
		return errors.New("TODO: invalid memo")
		// return ErrInvalidMemo("Memo too long")
	}
	if err := validateAmount(m.Amount); err != nil {
		return err
	}
	return validatePermissions(m.Arbiter, m.Sender, m.Recipient)
}

// Validate makes sure that this is sensible
func (m *ReleaseEscrowMsg) Validate() error {
	if m.Amount == nil {
		return nil
	}
	return validateAmount(m.Amount)
}

// Validate always returns true for no data
func (m *ReturnEscrowMsg) Validate() error {
	return nil
}

// Validate makes sure any included items are valid permissions
// and there is at least one change
func (m *UpdateEscrowPartiesMsg) Validate() error {
	if m.Arbiter == nil &&
		m.Sender == nil &&
		m.Recipient == nil {
		return errors.New("TODO: no parties included")
	}
	return validatePermissions(m.Arbiter, m.Sender, m.Recipient)
}

// validatePermissions returns an error if any permission doesn't validate
// nil is considered valid here
func validatePermissions(perms ...weave.Permission) error {
	for _, p := range perms {
		if p != nil {
			if err := p.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateAmount(amount []*x.Coin) error {
	// TODO: validate list of amounts... ugh []*x.Coin, not []x.Coin....
	return nil
	//   amt := s.GetAmount()
	// if x.IsEmpty(amt) || !amt.IsPositive() {
	//   return ErrInvalidAmount("Non-positive SendMsg")
	// }
}
