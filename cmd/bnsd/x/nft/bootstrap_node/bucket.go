package bootstrap_node

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/nft"
)

const (
	BucketName = "bsnnft"
)

type Bucket struct {
	orm.Bucket
}

func NewBucket() Bucket {
	return Bucket{
		Bucket: nft.WithOwnerIndex(orm.NewBucket(BucketName, NewBootstrapNodeToken(nil, nil, nil))),
	}
}

func NewBootstrapNodeToken(key []byte, owner weave.Address, approvals []nft.ActionApprovals) *orm.SimpleObj {
	return orm.NewSimpleObj(key, &BootstrapNodeToken{
		Base:    nft.NewNonFungibleToken(key, owner, approvals),
		Details: &TokenDetails{},
	})
}

func (b Bucket) Create(db weave.KVStore, owner weave.Address, id []byte,
	approvals []nft.ActionApprovals, blockchainID []byte, uri URI) (orm.Object, error) {
	obj, err := b.Get(db, id)
	switch {
	case err != nil:
		return nil, err
	case obj != nil:
		return nil, orm.ErrUniqueConstraint("id exists already")
	}
	obj = NewBootstrapNodeToken(id, owner, approvals)
	node, err := AsNode(obj)
	if err != nil {
		return nil, err
	}

	if err := node.SetUri(owner, uri); err != nil {
		return nil, err
	}

	return obj, node.SetBlockchainID(owner, blockchainID)
}
