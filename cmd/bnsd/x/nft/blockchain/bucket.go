package blockchain

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
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

func (b Bucket) Create(db weave.KVStore, owner weave.Address, id []byte, approvals []nft.ActionApprovals, chain Chain, iov IOV) (orm.Object, error) {
	obj, err := b.Get(db, id)
	switch {
	case err != nil:
		return nil, err
	case obj != nil:
		return nil, errors.DuplicateErr.New("id exists already")
	}
	obj = NewBlockchainToken(id, owner, approvals)

	blockChain, err := AsBlockchain(obj)
	if err != nil {
		return nil, err
	}

	if err := blockChain.SetChain(owner, chain); err != nil {
		return obj, err
	}

	return obj, blockChain.SetIov(owner, iov)
}
