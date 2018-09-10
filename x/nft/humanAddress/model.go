package humanAddress

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/nft"
	"github.com/pkg/errors"
)

const (
	// BucketName is where we store the nfts
	BucketName = "usrnft"
	// IndexName is the index to query nft by owner
	IndexName = "owner"
)

type HumanAddress interface {
	nft.BaseNFT
	GetPubKey() []byte
	SetPubKey([]byte)
}
type humanAddressNftAdapter struct {
	*nft.NonFungibleToken
}

func (a *humanAddressNftAdapter) GetPubKey() []byte {
	if a.GetHumanAddress() == nil {
		return nil
	}
	return a.GetHumanAddress().Account
}

func (a *humanAddressNftAdapter) SetPubKey(pubKey []byte) {
	a.Payload = &nft.NonFungibleToken_HumanAddress{
		HumanAddress: &nft.HumanAddressPayload{
			Account: pubKey,
		},
	}
}

// As HumanAddress will safely type-cast any value from Bucket
func AsHumanAddress(obj orm.Object) (HumanAddress, error) {
	if obj == nil || obj.Value() == nil {
		return nil, nil
	}
	x, ok := obj.Value().(*nft.NonFungibleToken)
	if !ok {
		return nil, errors.New("unsupported type")
	}
	return &humanAddressNftAdapter{x}, nil
}

// Bucket is a type-safe wrapper around orm.Bucket
type Bucket struct {
	orm.Bucket
}

//var _ nft.BaseBucket = Bucket{}

func NewBucket() Bucket {
	return Bucket{
		Bucket: orm.NewBucket(BucketName, nft.NewNonFungibleToken(nil)).WithIndex(IndexName, ownerIndex, false),
	}
}

// simple indexer for Wallet name
func ownerIndex(obj orm.Object) ([]byte, error) {
	if obj == nil {
		return nil, orm.ErrInvalidIndex("nil")
	}
	humanAddress, err := AsHumanAddress(obj)
	if err != nil {
		return nil, orm.ErrInvalidIndex("not human address")
	}
	// big-endian encoded int64
	return []byte(humanAddress.OwnerAddress()), nil
}

func (b Bucket) Create(db weave.KVStore, key []byte, pubKey []byte) (orm.Object, error) {
	obj, err := b.Get(db, key)
	switch {
	case err != nil:
		return nil, err
	case obj != nil:
		return nil, errors.New("key exists already") // todo: move into errors file
	}
	obj = nft.NewNonFungibleToken(key)
	humanAddress, err := AsHumanAddress(obj)
	if err != nil {
		return nil, err
	}
	humanAddress.SetPubKey(pubKey)
	return obj, nil
}
