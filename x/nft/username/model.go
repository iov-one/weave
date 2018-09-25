package username

import (
	"bytes"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/nft"
)

type Token interface {
	nft.BaseNFT
	GetChainAddresses() []ChainAddress
	SetChainAddresses(actor weave.Address, newKeys []ChainAddress) error
}

func (u *UsernameToken) Approvals() *nft.ApprovalOps {
	return u.Base.Approvals()
}

func (u *UsernameToken) GetChainAddresses() []ChainAddress {
	if u.Details == nil {
		return nil
	}
	return u.Details.Addresses
}
func (u *UsernameToken) SetChainAddresses(actor weave.Address, newAddresses []ChainAddress) error {
	// todo: this should be a sorted list

	if !u.OwnerAddress().Equals(actor) {
		panic("Not implemented, yet")
		// TODO: handle permissions
		//if !u.Base.HasApproval(actor, nft.ActionKindUpdateDetails) {
		//	return errors.ErrUnauthorized()
		//}
	}
	if containsDuplicateChains(newAddresses) {
		return nft.ErrDuplicateEntry()
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
		return errors.ErrInternal("must not be nil")
	}
	if containsDuplicateChains(t.Addresses) {
		return nft.ErrDuplicateEntry()
	}
	for _, k := range t.Addresses {
		if err := k.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func containsDuplicateChains(addresses []ChainAddress) bool {
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
	if len(p.ChainID) == 0 || len(p.ChainID) > 255 { // todo: take from blockchain id
		return nft.ErrInvalidLength()
	}
	switch l := len(p.Address); {
	case l == 0: // address can be empty
	case l < 12 || l > 50:
		return nft.ErrInvalidLength()
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
		return nil, nft.ErrUnsupportedTokenType()
	}
	return x, nil
}
