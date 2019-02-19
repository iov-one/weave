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

func WithOwnerIndex(bucket orm.Bucket) orm.Bucket {
	return bucket.WithIndex(OwnerIndexName, ownerIndex, false)
}

func ownerIndex(obj orm.Object) ([]byte, error) {
	if obj == nil {
		return nil, orm.ErrInvalidIndex.New("nil")
	}
	o, ok := obj.Value().(Owned)
	if !ok {
		return nil, orm.ErrInvalidIndex.New("unsupported type")
	}
	return []byte(o.OwnerAddress()), nil
}
