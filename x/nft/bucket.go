package nft

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
)

// OwnerIndexName is the index to query nft by owner
const OwnerIndexName = "owner"

type Owned interface {
	OwnerAddress() weave.Address
}

type BaseBucket interface {
	Issue(id []byte, owner weave.Address, initialPayload Payload) BaseNFT
	Load(id []byte) BaseNFT
	Revoke(id []byte)
}

type PersistentBaseBucket struct {
	orm.Bucket
}

func WithOwnerIndex(bucket orm.Bucket) orm.Bucket {
	return bucket.WithIndex(OwnerIndexName, ownerIndex, false)
}

func ownerIndex(obj orm.Object) ([]byte, error) {
	if obj == nil {
		return nil, orm.ErrInvalidIndex("nil")
	}
	o, ok := obj.Value().(Owned)
	if !ok {
		return nil, orm.ErrInvalidIndex("unsupported type")
	}
	// big-endian encoded int64
	return []byte(o.OwnerAddress()), nil
}
