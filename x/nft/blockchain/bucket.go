package blockchain

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/nft"
)

const (
	BucketName = "bchnft"
)

type Bucket struct {
	orm.Bucket
}

func NewBucket() Bucket {
	return Bucket{
		Bucket: nft.WithOwnerIndex(orm.NewBucket(BucketName, NewBlockchainToken(nil, nil, nil))),
	}
}

func NewBlockchainToken(key []byte, owner weave.Address, approvals []nft.ActionApprovals) *orm.SimpleObj {
	return orm.NewSimpleObj(key, &BlockchainToken{
		Base:    nft.NewNonFungibleToken(key, owner, approvals),
		Details: &TokenDetails{},
	})
}

func (b Bucket) Create(db weave.KVStore, owner weave.Address, id []byte, approvals []nft.ActionApprovals, networks []Network) (orm.Object, error) {
	obj, err := b.Get(db, id)
	switch {
	case err != nil:
		return nil, err
	case obj != nil:
		return nil, orm.ErrUniqueConstraint("id exists already")
	}
	obj = NewBlockchainToken(id, owner, approvals)

	humanAddress, err := AsBlockchain(obj)
	if err != nil {
		return nil, err
	}
	_ = humanAddress // todo set payload proper
	//return obj, humanAddress.SetNetworks(owner, networks)
	return obj, nil
}
