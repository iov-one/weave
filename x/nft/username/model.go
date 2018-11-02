package username

import (
	"bytes"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/nft"
	"github.com/iov-one/weave/x/nft/blockchain"
)

const (
	BucketName            = "usrnft"
	ChainAddressIndexName = "chainaddr"
	chainAddressSeparator = "*"
)

type UsernameTokenBucket struct {
	orm.Bucket
}

func NewUsernameTokenBucket() UsernameTokenBucket {
	bucket := orm.NewBucket(BucketName,
		orm.NewSimpleObj(nil, new(UsernameToken)))
	return UsernameTokenBucket{
		Bucket: bucket,
	}
}

// func NewUsernameTokenBucket() UsernameTokenBucket {
// 	return UsernameTokenBucket{
// 		Bucket: nft.WithOwnerIndex(orm.NewBucket(BucketName, orm.NewSimpleObj(nil, new(UsernameToken))).
// 			WithMultiKeyIndex(ChainAddressIndexName, chainAddressIndexer, true)),
// 	}
// }

func chainAddressIndexer(obj orm.Object) ([][]byte, error) {
	if obj == nil {
		return nil, orm.ErrInvalidIndex("nil")
	}
	u, err := AsUsername(obj)
	if err != nil {
		return nil, orm.ErrInvalidIndex("unsupported type")
	}
	idx := make([][]byte, 0, len(u.Addresses))
	for _, addr := range u.Addresses {
		idx = append(idx, bytes.Join([][]byte{addr.Address, addr.ChainID}, []byte(chainAddressSeparator)))
	}
	return idx, nil
}

var _ orm.CloneableData = (*UsernameToken)(nil)

func (u *UsernameToken) Validate() error {
	return nil
	// return u.Details.Validate()
}

func (u *UsernameToken) Copy() orm.CloneableData {
	return &UsernameToken{
		Id:        u.Id,
		Owner:     u.Owner,
		Addresses: u.Addresses,
		Approvals: u.Approvals,
	}
}

func containsDuplicateChains(addresses []*ChainAddress) bool {
	m := make(map[string]struct{})
	for _, k := range addresses {
		if _, ok := m[string(k.ChainID)]; ok {
			return true
		}
		m[string(k.ChainID)] = struct{}{}
	}
	return false
}

func (p ChainAddress) Equals(o ChainAddress) bool {
	return bytes.Equal(p.Address, o.Address) && bytes.Equal(p.ChainID, o.ChainID)
}

func (p *ChainAddress) Validate() error {
	if !blockchain.IsValidID(string(p.ChainID)) {
		return nft.ErrInvalidID()
	}
	switch l := len(p.Address); {
	case l < 12 || l > 50:
		return nft.ErrInvalidLength()
	}
	return nil
}

// AsUsername will safely type-cast any value from Bucket
func AsUsername(obj orm.Object) (*UsernameToken, error) {
	if obj == nil || obj.Value() == nil {
		return nil, nil
	}
	x, ok := obj.Value().(*UsernameToken)
	if !ok {
		return nil, nft.ErrUnsupportedTokenType()
	}
	return x, nil
}

func validateID(i nft.Identified) error {
	if i == nil {
		return errors.ErrInternal("must not be nil")
	}
	if !isValidID(string(i.GetId())) {
		return nft.ErrInvalidID()
	}
	return nil
}
