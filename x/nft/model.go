package nft

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
)

var _ orm.CloneableData = (*NonFungibleToken)(nil)

func (m *NonFungibleToken) Validate() error {
	var validation *Validation
	if !validation.IsValidTokenID(m.Id) {
		return ErrInvalidID()
	}

	if err := weave.Address(m.Owner).Validate(); err != nil {
		return err
	}
	// TODO: impl proper validation
	//for _, a := range m.ActionApprovals {
	//if err := a.Validate(); err != nil {
	//	return err
	//}
	//}
	return nil
}

func (m *NonFungibleToken) Copy() orm.CloneableData {
	return m.Clone()
}

func (m *NonFungibleToken) Clone() *NonFungibleToken {
	actionApprovals := make([]ActionApprovals, len(m.ActionApprovals))
	for i, v := range m.ActionApprovals {
		actionApprovals[i] = v.Clone()
	}
	return &NonFungibleToken{
		Id:              m.Id,
		Owner:           m.Owner,
		ActionApprovals: actionApprovals,
	}
}

func NewNonFungibleToken(key []byte, owner weave.Address) *NonFungibleToken {
	return &NonFungibleToken{
		Id:    key,
		Owner: owner,
	}
}

func (u *NonFungibleToken) OwnerAddress() weave.Address {
	return weave.Address(u.Owner)
}

func (m *NonFungibleToken) Approvals() *ApprovalOps {
	return NewApprovalOps(m.OwnerAddress(), &m.ActionApprovals)
}

func (m *NonFungibleToken) HasApproval(actor weave.Address, action string) bool {
	return !NewApprovalOps(m.OwnerAddress(), &m.ActionApprovals).
		List().ForAction(action).ForAddress(actor).IsEmpty()
}

type BaseNFT interface {
	Owned
	//GetId() []byte
	Approvals() *ApprovalOps
	//Transfer(newOwner weave.Address) error
}

//TODO: Better name
type Identified interface {
	GetId() []byte
}
