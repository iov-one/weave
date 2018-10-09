package username

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/nft"
)

const (
	BucketName = "usrnft"
)

type Bucket struct {
	orm.Bucket
}

func NewBucket() Bucket {
	return Bucket{
		Bucket: nft.WithOwnerIndex(orm.NewBucket(BucketName, NewUsernameToken(nil, nil))),
	}
}

func NewUsernameToken(key []byte, owner weave.Address) *orm.SimpleObj {
	return orm.NewSimpleObj(key, &UsernameToken{
		Base:    nft.NewNonFungibleToken(key, owner),
		Details: &TokenDetails{},
	})
}

func (b Bucket) Create(db weave.KVStore, owner weave.Address, id []byte, addresses []ChainAddress) (orm.Object, error) {
	obj, err := b.Get(db, id)
	switch {
	case err != nil:
		return nil, err
	case obj != nil:
		return nil, orm.ErrUniqueConstraint("id exists already")
	}
	obj = NewUsernameToken(id, owner)
	humanAddress, err := AsUsername(obj)
	if err != nil {
		return nil, err
	}
	return obj, humanAddress.SetChainAddresses(owner, addresses)
}
