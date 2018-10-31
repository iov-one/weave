package username

import (
	"bytes"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/nft"
)

const (
	BucketName            = "usrnft"
	ChainAddressIndexName = "chainaddr"
	chainAddressSeparator = "*"
)

type Bucket struct {
	orm.Bucket
}

func NewBucket() Bucket {
	return Bucket{
		Bucket: nft.WithOwnerIndex(orm.NewBucket(BucketName, NewUsernameToken(nil, nil, nil))).
			WithMultiKeyIndex(ChainAddressIndexName, chainAddressIndexer, true),
	}
}

func NewUsernameToken(key []byte, owner weave.Address, approvals []nft.ActionApprovals) *orm.SimpleObj {
	return orm.NewSimpleObj(key, &UsernameToken{
		Base:    nft.NewNonFungibleToken(key, owner, approvals),
		Details: &TokenDetails{},
	})
}

func (b Bucket) Create(db weave.KVStore, owner weave.Address, id []byte, approvals []nft.ActionApprovals, addresses []ChainAddress) (orm.Object, error) {
	obj, err := b.Get(db, id)
	switch {
	case err != nil:
		return nil, err
	case obj != nil:
		return nil, orm.ErrUniqueConstraint("id exists already")
	}
	obj = NewUsernameToken(id, owner, approvals)
	humanAddress, err := AsUsername(obj)
	if err != nil {
		return nil, err
	}
	// height is not relevant when creating the token
	return obj, humanAddress.SetChainAddresses(-1, owner, addresses)
}

func chainAddressIndexer(obj orm.Object) ([][]byte, error) {
	if obj == nil {
		return nil, orm.ErrInvalidIndex("nil")
	}
	u, err := AsUsername(obj)
	if err != nil {
		return nil, orm.ErrInvalidIndex("unsupported type")
	}
	idx := make([][]byte, 0, len(u.GetChainAddresses()))
	for _, addr := range u.GetChainAddresses() {
		idx = append(idx, bytes.Join([][]byte{addr.Address, addr.ChainID}, []byte(chainAddressSeparator)))
	}
	return idx, nil
}
