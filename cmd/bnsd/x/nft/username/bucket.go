package username

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/nft"
)

type Bucket struct {
	orm.Bucket
}

func NewBucket() Bucket {
	t := NewUsernameToken(nil, nil, nil)
	b := orm.NewBucket("usrnft", t)
	return Bucket{
		Bucket: nft.WithOwnerIndex(b),
	}
}

func NewUsernameToken(key []byte, owner weave.Address, approvals []nft.ActionApprovals) *orm.SimpleObj {
	return orm.NewSimpleObj(key, &UsernameToken{
		Metadata: &weave.Metadata{Schema: 1},
		Base:     nft.NewNonFungibleToken(key, owner, approvals),
		Details:  &TokenDetails{},
	})
}

func (b Bucket) Create(db weave.KVStore, owner weave.Address, id []byte, approvals []nft.ActionApprovals, addresses []ChainAddress) (orm.Object, error) {
	obj, err := b.Get(db, id)
	switch {
	case err != nil:
		return nil, err
	case obj != nil:
		return nil, errors.Wrap(errors.ErrDuplicate, "id exists already")
	}
	obj = NewUsernameToken(id, owner, approvals)
	humanAddress, err := AsUsername(obj)
	if err != nil {
		return nil, err
	}
	// height is not relevant when creating the token
	return obj, humanAddress.SetChainAddresses(owner, addresses)
}
