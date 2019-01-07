package nft

import (
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
	if _, ok := validActions[m.Action]; !ok {
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
	if _, ok := validActions[m.Action]; !ok {
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

// validActions is an index of all available and supported by the
// implementation actions. This is to be used to validate if requested action
// is valid and can be handled.
var validActions = map[Action]struct{}{
	UpdateDetails:   struct{}{},
	Transfer:        struct{}{},
	UpdateApprovals: struct{}{},
}
