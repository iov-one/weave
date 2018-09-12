package blockchain

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/nft"
	"github.com/pkg/errors"
)

const (
	// BucketName is where we store the nfts
	BucketName = "blknft"
)

type BlockchainNFT interface {
	nft.BaseNFT
	UpdateDetails(actor weave.Address, details TokenDetails) error
	GetDetails() *TokenDetails
}

func (b *BlockchainToken) OwnerAddress() weave.Address {
	return weave.Address(b.Base.Owner)
}

func (b *BlockchainToken) Approvals() *nft.ApprovalOperations {
	return nft.NewApprovalOperations(b, &b.Base.Approvals)
}

func (b *BlockchainToken) Transfer(newOwner weave.Address) error {
	// todo: anything special to check?
	return b.Base.Transfer(newOwner)
}

func (b *BlockchainToken) UpdateDetails(actor weave.Address, newDetails TokenDetails) error {
	if !b.OwnerAddress().Equals(actor) {
		if !b.Base.HasApproval(actor, nft.ActionKind_UpdateDetails) {
			return errors.New("unauthorized")
		}
	}
	b.Details = &newDetails
	return nil
}

func (b *BlockchainToken) Validate() error {
	// todo: impl
	return b.OwnerAddress().Validate()
}

func (b *BlockchainToken) Copy() orm.CloneableData {
	// todo: impl
	return &BlockchainToken{
		Base:    b.Base,
		Details: b.Details,
	}
}

// As BlockchainNFT will safely type-cast any value from Bucket
func AsBlockchainNFT(obj orm.Object) (BlockchainNFT, error) {
	if obj == nil || obj.Value() == nil {
		return nil, nil
	}
	x, ok := obj.Value().(*BlockchainToken)
	if !ok {
		return nil, errors.New("unsupported type") // todo: move
	}
	return x, nil
}

// Bucket is a type-safe wrapper around orm.Bucket
type Bucket struct {
	orm.Bucket
}

func NewBucket() Bucket {
	return Bucket{
		Bucket: nft.WithOwnerIndex(orm.NewBucket(BucketName, NewBlockchainToken(nil, nil))),
	}
}
func NewBlockchainToken(key []byte, owner weave.Address) *orm.SimpleObj {
	token := nft.NewNonFungibleToken(key, owner)
	return orm.NewSimpleObj(key, &BlockchainToken{
		Base:    token,
		Details: &TokenDetails{},
	})
}

func (b Bucket) Create(db weave.KVStore, owner weave.Address, key []byte, details TokenDetails) (orm.Object, error) {
	obj, err := b.Get(db, key)
	switch {
	case err != nil:
		return nil, err
	case obj != nil:
		return nil, errors.New("key exists already") // todo: move into errors file
	}
	obj = NewBlockchainToken(key, owner)
	bc, err := AsBlockchainNFT(obj)
	if err != nil {
		return nil, err
	}
	return obj, bc.UpdateDetails(owner, details)
}
