package blockchain

import (
	"encoding/json"
	"regexp"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/nft"
)

var (
	//todo: revisit pattern
	IsValidCodec = regexp.MustCompile(`^[a-zA-Z0-9_.]{3,20}$`).MatchString
)

type Token interface {
	nft.BaseNFT
	GetChain() Chain
	GetIov() IOV
	SetChain(actor weave.Address, chain Chain) error
	SetIov(actor weave.Address, iov IOV) error
}

func (m *BlockchainToken) OwnerAddress() weave.Address {
	return m.Base.OwnerAddress()
}

func (m *BlockchainToken) Approvals() *nft.ApprovalOps {
	return m.Base.Approvals()
}

func (m *BlockchainToken) SetApprovals(a nft.Approvals) {
	m.Base.ActionApprovals = a.AsPersistable()
}

func (m *BlockchainToken) GetChain() Chain {
	return m.Details.Chain
}

func (m *BlockchainToken) GetIov() IOV {
	return m.Details.Iov
}

func (m *BlockchainToken) SetChain(actor weave.Address, chain Chain) error {
	if !m.OwnerAddress().Equals(actor) {
		panic("Not implemented, yet")
		// TODO: handle permissions
	}

	m.Details.Chain = chain
	return nil
}

func (m *BlockchainToken) SetIov(actor weave.Address, iov IOV) error {
	if !m.OwnerAddress().Equals(actor) {
		panic("Not implemented, yet")
		// TODO: handle permissions
	}

	m.Details.Iov = iov
	return nil
}

func (m *BlockchainToken) Transfer(newOwner weave.Address) error {
	panic("implement me")
}

func (m *BlockchainToken) Validate() error {
	if err := m.Base.Validate(); err != nil {
		return err
	}
	if err := m.Approvals().List().Validate(); err != nil {
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
	return &TokenDetails{Chain: m.Chain, Iov: m.Iov}
}

func (m *TokenDetails) Validate() error {
	if m == nil {
		return errors.ErrInternal("must not be nil")
	}
	if err := m.Iov.Validate(); err != nil {
		return err
	}
	return m.Chain.Validate()
}

func (m IOV) Validate() error {
	if !IsValidCodec(m.Codec) {
		return nft.ErrInvalidCodec(m.Codec)
	}

	if m.CodecConfig != "" {
		var js interface{}
		bytes := []byte(m.CodecConfig)
		if err := json.Unmarshal(bytes, &js); err != nil {
			return nft.ErrInvalidJson()
		}
	}

	return nil
}

func (m Chain) Validate() error {
	// todo: impl
	// ticker id already validated on issue etc via bucket
	// decide on the validation rules here
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
