package username

import (
	"bytes"
	"regexp"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/nft"
)

// ModelName is used to recognize the model provided by this NFT implementation.
// It can be used by the nft/base extension to register a handler.
const ModelName = "username"

var validBlockchainID = regexp.MustCompile(`^[a-zA-Z0-9_.-]{4,32}$`).Match

type Token interface {
	nft.BaseNFT
	GetChainAddresses() []ChainAddress
	SetChainAddresses(actor weave.Address, newKeys []ChainAddress) error
}

func (u *UsernameToken) Approvals() *nft.ApprovalOps {
	return u.Base.Approvals()
}

func (m *UsernameToken) SetApprovals(a nft.Approvals) {
	m.Base.ActionApprovals = a.AsPersistable()
}

func (u *UsernameToken) GetChainAddresses() []ChainAddress {
	if u.Details == nil {
		return nil
	}
	return u.Details.Addresses
}

func (u *UsernameToken) SetChainAddresses(actor weave.Address, newAddresses []ChainAddress) error {
	dup := containsDuplicateChains(newAddresses)
	if dup != nil {
		return errors.Wrapf(errors.ErrDuplicate, "chain %s", nft.PrintableID(dup))
	}
	u.Details = &TokenDetails{Addresses: newAddresses}
	return nil
}

func (u *UsernameToken) OwnerAddress() weave.Address {
	return u.Base.OwnerAddress()
}

func (u *UsernameToken) Transfer(newOwner weave.Address) error {
	panic("implement me")
}

func (u *UsernameToken) Validate() error {
	if err := u.Base.Validate(); err != nil {
		return err
	}
	if err := u.Approvals().List().Validate(); err != nil {
		return err
	}
	return u.Details.Validate()
}

func (u *UsernameToken) Copy() orm.CloneableData {
	return &UsernameToken{
		Base:    u.Base.Clone(),
		Details: u.Details.Clone(),
	}
}

func (t *TokenDetails) Clone() *TokenDetails {
	a := make([]ChainAddress, len(t.Addresses))
	for i, v := range t.Addresses {
		a[i] = v
	}
	return &TokenDetails{Addresses: a}
}

func (t *TokenDetails) Validate() error {
	if t == nil {
		return errors.Wrap(errors.ErrInput, "token details must not be nil")
	}
	dup := containsDuplicateChains(t.Addresses)
	if dup != nil {
		return errors.Wrapf(errors.ErrDuplicate, "chain %s", nft.PrintableID(dup))
	}
	for _, k := range t.Addresses {
		if err := k.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// returns the duplicated chainId or nil if no duplicates
func containsDuplicateChains(addresses []ChainAddress) []byte {
	m := make(map[string]struct{})
	for _, k := range addresses {
		if _, ok := m[string(k.BlockchainID)]; ok {
			return k.BlockchainID
		}
		m[string(k.BlockchainID)] = struct{}{}
	}
	return nil
}

func (p ChainAddress) Equals(o ChainAddress) bool {
	return p.Address == o.Address && bytes.Equal(p.BlockchainID, o.BlockchainID)
}

func (p *ChainAddress) Validate() error {
	if !validBlockchainID(p.BlockchainID) {
		return errors.Wrapf(errors.ErrInput, "id: %s", nft.PrintableID(p.BlockchainID))
	}
	if n := len(p.Address); n < 2 || n > 50 {
		return errors.Wrapf(errors.ErrInput, "address length: %s", p.Address)
	}
	return nil
}

// AsUsername will safely type-cast any value from Bucket
func AsUsername(obj orm.Object) (Token, error) {
	if obj == nil || obj.Value() == nil {
		return nil, nil
	}
	x, ok := obj.Value().(*UsernameToken)
	if !ok {
		return nil, errors.Wrap(errors.ErrInput, nft.UnsupportedTokenType)
	}
	return x, nil
}
