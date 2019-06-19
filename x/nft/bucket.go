package nft

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
)

// OwnerIndexName is the index to query nft by owner
const OwnerIndexName = "owner"

type Owned interface {
	OwnerAddress() weave.Address
}

func WithOwnerIndexOpt() orm.BucketBuilderOpt {
	return orm.BucketBuilderOptFunc(
		func(bb *orm.BucketBuilder) *orm.BucketBuilder {
			return bb.WithIndex(OwnerIndexName, ownerIndex, false)
		})
}

func ownerIndex(obj orm.Object) ([]byte, error) {
	if obj == nil {
		return nil, errors.Wrap(orm.ErrInvalidIndex, "nil")
	}
	o, ok := obj.Value().(Owned)
	if !ok {
		return nil, errors.Wrap(orm.ErrInvalidIndex, "unsupported type")
	}
	return []byte(o.OwnerAddress()), nil
}
