package username

import (
	"regexp"

	"github.com/iov-one/weave"
)

var _ weave.Msg = (*CreateUsernameTokenMsg)(nil)

const (
	pathIssueTokenMsg    = "nft/username/issue"
	pathAddAddressMsg    = "nft/username/address/add"
	pathRemoveAddressMsg = "nft/username/address/remove"
)

var (
	isValidID = regexp.MustCompile(`^[a-z0-9\.,\+\-_@]{4,64}$`).MatchString
)

// Path fulfills weave.Msg interface to allow routing
func (CreateUsernameTokenMsg) Path() string {
	return pathIssueTokenMsg
}

// Path returns the routing path for this message
func (*AddChainAddressMsg) Path() string {
	return pathAddAddressMsg
}

// Path returns the routing path for this message
func (*RemoveChainAddressMsg) Path() string {
	return pathRemoveAddressMsg
}

func (m *CreateUsernameTokenMsg) Validate() error {
	if err := validateID(m); err != nil {
		return err
	}
	// if err := m.Details.Validate(); err != nil {
	// 	return err
	// }

	addr := weave.Address(m.Owner)

	if err := addr.Validate(); err != nil {
		return err
	}

	return nil
}

func (m *AddChainAddressMsg) Validate() error {
	if err := validateID(m); err != nil {
		return err
	}
	return m.Addresses.Validate()
}

func (m *RemoveChainAddressMsg) Validate() error {
	if err := validateID(m); err != nil {
		return err
	}
	return m.Addresses.Validate()
}
