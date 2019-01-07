package nft

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
)

var _ orm.CloneableData = (*NonFungibleToken)(nil)

func (m *NonFungibleToken) Validate() error {
	if !isValidTokenID(m.ID) {
		return ErrInvalidID(m.ID)
	}

	if err := weave.Address(m.Owner).Validate(); err != nil {
		return err
	}

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
		ID:              m.ID,
		Owner:           m.Owner,
		ActionApprovals: actionApprovals,
	}
}

func NewNonFungibleToken(key []byte, owner weave.Address, approvals []ActionApprovals) *NonFungibleToken {
	return &NonFungibleToken{
		ID:              key,
		Owner:           owner,
		ActionApprovals: approvals,
	}
}

func (u *NonFungibleToken) OwnerAddress() weave.Address {
	return weave.Address(u.Owner)
}

func (m *NonFungibleToken) Approvals() *ApprovalOps {
	return NewApprovalOps(m.OwnerAddress(), &m.ActionApprovals)
}

func (m *NonFungibleToken) SetApprovals(a Approvals) {
	m.ActionApprovals = a.AsPersistable()
}

func (m *NonFungibleToken) HasApproval(actor weave.Address, action Action) bool {
	return !NewApprovalOps(m.OwnerAddress(), &m.ActionApprovals).
		List().ForAction(action).ForAddress(actor).IsEmpty()
}

type BaseNFT interface {
	Owned
	//GetId() []byte
	Approvals() *ApprovalOps
	//Set new approvals
	SetApprovals(Approvals)
}

//TODO: Better name
type Identified interface {
	GetID() []byte
}
