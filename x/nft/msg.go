package nft

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

const(
	pathAddApprovalMsg    = "nft/approval/add"
	pathRemoveApprovalMsg = "nft/approval/remove"
)

func (*AddApprovalMsg) Path() string {
	return pathAddApprovalMsg
}

func (*RemoveApprovalMsg) Path() string {
	return pathRemoveApprovalMsg
}

func (m AddApprovalMsg) Validate() error {
	var validation *Validation
	if err := weave.Address(m.Address).Validate(); err != nil {
		return err
	}
	if !validation.IsValidAction(m.Action) {
		return errors.ErrInternal("action must be valid")
	}
	if !validation.IsValidTokenID(m.Id) {
		return errors.ErrInternal("id must be valid")
	}
	return m.Options.Validate()
}

func (m RemoveApprovalMsg) Validate() error {
	var validation *Validation
	if err := weave.Address(m.Address).Validate(); err != nil {
		return err
	}
	if !validation.IsValidAction(m.Action) {
		return errors.ErrInternal("action must be valid")
	}
	if !validation.IsValidTokenID(m.Id) {
		return errors.ErrInternal("id must be valid")
	}
	return nil
}
