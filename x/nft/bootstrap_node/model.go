package bootstrap_node

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/nft"
	"github.com/iov-one/weave/x/nft/blockchain"
)

//TODO: This is very similar to ticker token, I bet we can consolidate those somehow
// or at least their common parts
type Token interface {
	nft.BaseNFT
	GetBlockchainID() []byte
	SetBlockchainID(actor weave.Address, id []byte) error
	SetUri(actor weave.Address, uri string) error
	GetUri() string
}

func (m *BootstrapNodeToken) OwnerAddress() weave.Address {
	return m.Base.OwnerAddress()
}

func (m *BootstrapNodeToken) GetBlockchainID() []byte {
	return m.Details.BlockchainID
}

func (m *BootstrapNodeToken) GetUri() URI {
	return m.Details.Uri
}

func (m *BootstrapNodeToken) Approvals() *nft.ApprovalOps {
	return m.Base.Approvals()
}

func (m *BootstrapNodeToken) SetUri(actor weave.Address, uri string) error {
	if !m.OwnerAddress().Equals(actor) {
		panic("Not implemented, yet")
		// TODO: handle permissions
	}

	m.Details.Uri = uri
	return nil
}

func (m *BootstrapNodeToken) SetBlockchainID(actor weave.Address, id []byte) error {
	if !m.OwnerAddress().Equals(actor) {
		panic("Not implemented, yet")
		// TODO: handle permissions
	}

	newID := make([]byte, len(id))
	_ = copy(newID, id)

	m.Details.BlockchainID = newID
	return nil
}

func (m *BootstrapNodeToken) Transfer(newOwner weave.Address) error {
	panic("implement me")
}

func (m *BootstrapNodeToken) Validate() error {
	if err := m.Base.Validate(); err != nil {
		return err
	}
	if err := m.Approvals().List().Validate(); err != nil {
		return err
	}
	return m.Details.Validate()
}

func (m *BootstrapNodeToken) Copy() orm.CloneableData {
	return &BootstrapNodeToken{
		Base:    m.Base.Clone(),
		Details: m.Details.Clone(),
	}
}

func (m *TokenDetails) Clone() *TokenDetails {
	// todo: impl
	return &TokenDetails{BlockchainID: m.BlockchainID, Uri: m.Uri}
}

func (m *TokenDetails) Validate() error {
	//TODO: Validate uri
	if m == nil {
		return errors.ErrInternal("must not be nil")
	}
	if m.BlockchainID == nil || !blockchain.IsValidID(string(m.BlockchainID)) {
		return nft.ErrInvalidEntry()
	}
	return nil
}

// AsNode will safely type-cast any value from Bucket
func AsNode(obj orm.Object) (Token, error) {
	if obj == nil || obj.Value() == nil {
		return nil, nil
	}
	x, ok := obj.Value().(*BootstrapNodeToken)
	if !ok {
		return nil, nft.ErrUnsupportedTokenType()
	}
	return x, nil
}
