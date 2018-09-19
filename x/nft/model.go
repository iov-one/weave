package nft

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
)

var _ orm.CloneableData = (*NonFungibleToken)(nil)

const (
	minIDLength = 4
	maxIDLength = 256
)

func (m *NonFungibleToken) Validate() error {
	if len(m.Id) < minIDLength || len(m.Id) > maxIDLength {
		return ErrInvalidID()
	}
	if err := weave.Address(m.Owner).Validate(); err != nil {
		return err
	}
	for _, a := range m.ActionApprovals {
		if err := a.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (m *NonFungibleToken) Copy() orm.CloneableData {
	return m.Clone()
}

func (m *NonFungibleToken) Clone() *NonFungibleToken {
	if m == nil {
		return nil
	}
	actionApprovals := make([]*ActionApprovals, len(m.ActionApprovals))
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

type BaseNFT interface {
	Owned
	//	GetId() []byte
	//Approvals() *ApprovalOperations
	//Transfer(newOwner weave.Address) error
}
