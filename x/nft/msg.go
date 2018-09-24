package nft

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

var _ weave.Msg = (*AddApprovalMsg)(nil)
var _ weave.Msg = (*RemoveApprovalMsg)(nil)

const (
	pathAddApproval    = "nft/add-approval"
	pathRemoveApproval = "nft/remove-approval"
)

// Path returns the routing path for this message
func (*AddApprovalMsg) Path() string {
	return pathAddApproval
}

// Path returns the routing path for this message
func (*RemoveApprovalMsg) Path() string {
	return pathRemoveApproval
}

func (m *AddApprovalMsg) Validate() error {
	if err := weave.Address(m.Address).Validate(); err != nil {
		return err
	}
	if m.Action == "" {
		return errors.ErrInternal("action must not be empty")
	}
	//TODO: Figure out whether we need to incorporate same check as in NFT
	if len(m.Id) == 0 {
		return errors.ErrInternal("id must not be empty")
	}
	return nil
}

func (m *RemoveApprovalMsg) Validate() error {
	if err := weave.Address(m.Address).Validate(); err != nil {
		return err
	}
	if m.Action == "" {
		return errors.ErrInternal("action must not be empty")
	}
	//TODO: Figure out whether we need to incorporate same check as in NFT
	if len(m.Id) == 0 {
		return errors.ErrInternal("id must not be empty")
	}
	return nil
}
