package ticker

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/nft"
	"github.com/iov-one/weave/x/nft/blockchain"
)

type Token interface {
	nft.BaseNFT
	GetBlockchainID() []byte
	SetBlockchainID(actor weave.Address, id []byte) error
}

func (m *TickerToken) OwnerAddress() weave.Address {
	return m.Base.OwnerAddress()
}

func (m *TickerToken) GetBlockchainID() []byte {
	return m.Details.BlockchainID
}

func (m *TickerToken) Approvals() *nft.ApprovalOps {
	return m.Base.Approvals()
}

func (m *TickerToken) SetBlockchainID(actor weave.Address, id []byte) error {
	if !m.OwnerAddress().Equals(actor) {
		panic("Not implemented, yet")
		// TODO: handle permissions
	}

	newID := make([]byte, len(id))
	_ = copy(newID, id)

	m.Details.BlockchainID = newID
	return nil
}

func (m *TickerToken) Transfer(newOwner weave.Address) error {
	panic("implement me")
}

func (m *TickerToken) Validate() error {
	if err := m.Base.Validate(); err != nil {
		return err
	}
	if err := m.Approvals().List().Validate(); err != nil {
		return err
	}
	return m.Details.Validate()
}

func (m *TickerToken) Copy() orm.CloneableData {
	return &TickerToken{
		Base:    m.Base.Clone(),
		Details: m.Details.Clone(),
	}
}

func (m *TokenDetails) Clone() *TokenDetails {
	// todo: impl
	return &TokenDetails{BlockchainID: m.BlockchainID}
}

func (m *TokenDetails) Validate() error {
	if m == nil {
		return errors.ErrInternal("must not be nil")
	}
	if m.BlockchainID == nil || !blockchain.IsValidID(string(m.BlockchainID)) {
		return nft.ErrInvalidEntry()
	}
	return nil
}

// AsUsername will safely type-cast any value from Bucket
func AsTicker(obj orm.Object) (Token, error) {
	if obj == nil || obj.Value() == nil {
		return nil, nil
	}
	x, ok := obj.Value().(*TickerToken)
	if !ok {
		return nil, nft.ErrUnsupportedTokenType()
	}
	return x, nil
}
