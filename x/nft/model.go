package nft

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
)

var _ orm.CloneableData = (*NonFungibleToken)(nil)

func (m *NonFungibleToken) Validate() error {
	panic("implement me")
}

func (m *NonFungibleToken) Copy() orm.CloneableData {
	panic("implement me")
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
