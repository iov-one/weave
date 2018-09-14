package humanaddr

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/nft"
	"github.com/pkg/errors"
)

const (
	// BucketName is where we store the nfts
	BucketName = "usrnft"
)

func (a *TokenDetails) Clone() *TokenDetails {
	// todo: revisit to impl proper cloning
	x := *a
	return &x
}

type HumanAddress interface {
	nft.BaseNFT
	GetPubKey() []byte
	SetPubKey(weave.Address, []byte) error
}

func (a *HumanAddressToken) Approvals() *nft.ApprovalOperations {
	return nft.NewApprovalOperations(a, &a.Base.ActionApprovals)
}

func (a *HumanAddressToken) OwnerAddress() weave.Address {
	return weave.Address(a.Base.Owner)
}

func (a *HumanAddressToken) Transfer(newOwner weave.Address) error {
	// todo: anything special to check?
	return a.Base.Transfer(newOwner)
}

func (a *HumanAddressToken) GetPubKey() []byte {
	if a.Details == nil {
		return nil
	}
	return a.Details.PublicKey
}

func (a *HumanAddressToken) SetPubKey(actor weave.Address, newPubKey []byte) error {
	if !a.OwnerAddress().Equals(actor) {
		if !a.Base.HasApproval(actor, nft.ActionKind_UpdateDetails) {
			return errors.New("unauthorized")
		}
	}
	a.Details = &TokenDetails{PublicKey: newPubKey}
	return nil
}

func (a *HumanAddressToken) Validate() error {
	// todo: impl
	return a.OwnerAddress().Validate()
}

func (a *HumanAddressToken) Copy() orm.CloneableData {
	// todo: impl
	return &HumanAddressToken{
		Base:    a.Base,
		Details: a.Details.Clone(),
	}
}

// As HumanAddress will safely type-cast any value from Bucket
func AsHumanAddress(obj orm.Object) (HumanAddress, error) {
	if obj == nil || obj.Value() == nil {
		return nil, nil
	}
	x, ok := obj.Value().(*HumanAddressToken)
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
		Bucket: nft.WithOwnerIndex(orm.NewBucket(BucketName, NewHumanAddressToken(nil, nil))),
	}
}
func NewHumanAddressToken(key []byte, owner weave.Address) *orm.SimpleObj {
	return orm.NewSimpleObj(key, &HumanAddressToken{
		Base:    nft.NewNonFungibleToken(key, owner),
		Details: &TokenDetails{},
	})
}

func (b Bucket) Create(db weave.KVStore, owner weave.Address, key []byte, pubKey []byte) (orm.Object, error) {
	obj, err := b.Get(db, key)
	switch {
	case err != nil:
		return nil, err
	case obj != nil:
		return nil, errors.New("key exists already") // todo: move into errors file
	}
	obj = NewHumanAddressToken(key, owner)
	humanAddress, err := AsHumanAddress(obj)
	if err != nil {
		return nil, err
	}
	return obj, humanAddress.SetPubKey(owner, pubKey)
}
