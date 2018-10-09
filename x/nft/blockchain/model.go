package blockchain

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/nft"
)

type Token interface {
	nft.BaseNFT
	GetNetworks() []Network
}

func (m *BlockchainToken) OwnerAddress() weave.Address {
	return weave.Address(m.Base.Owner)
}
func (m *BlockchainToken) GetNetworks() []Network {
	return m.Details.Networks
}

func (m *BlockchainToken) Transfer(newOwner weave.Address) error {
	panic("implement me")
}

func (m *BlockchainToken) Validate() error {
	if err := m.Base.Validate(); err != nil {
		return err
	}
	return m.Details.Validate()
}

func (m *BlockchainToken) Copy() orm.CloneableData {
	return &BlockchainToken{
		Base:    m.Base.Clone(),
		Details: m.Details.Clone(),
	}
}

func (m *TokenDetails) Clone() *TokenDetails {
	// todo: impl
	return &TokenDetails{Networks: m.Networks}
}

func (m *TokenDetails) Validate() error {
	if m == nil {
		return errors.ErrInternal("must not be nil")
	}
	for _, v := range m.Networks {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (m *Network) Validate() error {
	// todo: impl
	return nil
}

// AsUsername will safely type-cast any value from Bucket
func AsBlockchain(obj orm.Object) (Token, error) {
	if obj == nil || obj.Value() == nil {
		return nil, nil
	}
	x, ok := obj.Value().(*BlockchainToken)
	if !ok {
		return nil, nft.ErrUnsupportedTokenType()
	}
	return x, nil
}
