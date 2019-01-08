package nft

import (
	"regexp"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

const (
	PathAddApprovalMsg    = "nft/approval/add"
	PathRemoveApprovalMsg = "nft/approval/remove"
)

type ApprovalMsg interface {
	GetT() string
	Identified
}

func (*AddApprovalMsg) Path() string {
	return PathAddApprovalMsg
}

func (*RemoveApprovalMsg) Path() string {
	return PathRemoveApprovalMsg
}

func (m AddApprovalMsg) Validate() error {
	if err := weave.Address(m.Address).Validate(); err != nil {
		return err
	}
	if !isValidAction(m.Action) {
		return errors.ErrInternal("invalid action")
	}
	if !isValidTokenID(m.ID) {
		return errors.ErrInternal("invalid token ID")
	}
	return m.Options.Validate()
}

func (m RemoveApprovalMsg) Validate() error {
	if err := weave.Address(m.Address).Validate(); err != nil {
		return err
	}
	if !isValidAction(m.Action) {
		return errors.ErrInternal("invalid action")
	}
	if !isValidTokenID(m.ID) {
		return errors.ErrInternal("invalid token ID")
	}
	return nil
}

// Action represents available and supported by the implementation actions.
// This is just a string type alias, but using it increase the clarity of the
// API.
type Action string

const (
	UpdateDetails   Action = "ActionUpdateDetails"
	Transfer        Action = "ActionTransfer"
	UpdateApprovals Action = "ActionUpdateApprovals"
)

// isValidAction returns true if given value is a valid action name. Action can
// be of type string or Action.
//
// Although all known to nft implementation actions are declared as constatns,
// user of nft might extend it with custom action strings. Because of this, we
// cannot validate the action by comparing to list of all known actions. We can
// only ensure that given name follows certain rules.
func isValidAction(action interface{}) bool {
	switch a := action.(type) {
	case Action:
		return isValidActionString(string(a))
	case string:
		return isValidActionString(a)
	default:
		return false
	}
}

var isValidActionString = regexp.MustCompile(`^[A-Za-z]{4,32}$`).MatchString
