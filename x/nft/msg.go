package nft

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &AddApprovalMsg{}, migration.NoModification)
	migration.MustRegister(1, &RemoveApprovalMsg{}, migration.NoModification)
}

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
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if err := m.Address.Validate(); err != nil {
		return err
	}
	if !isValidAction(m.Action) {
		return errors.Wrap(errors.ErrInput, "invalid action")
	}
	if !isValidTokenID(m.ID) {
		return errors.Wrap(errors.ErrInput, "invalid token ID")
	}
	return m.Options.Validate()
}

func (m RemoveApprovalMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if err := m.Address.Validate(); err != nil {
		return err
	}
	if !isValidAction(m.Action) {
		return errors.Wrap(errors.ErrInput, "invalid action")
	}
	if !isValidTokenID(m.ID) {
		return errors.Wrap(errors.ErrInput, "invalid token ID")
	}
	return nil
}
