package username

import (
	"regexp"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x/nft"
)

var _ weave.Msg = (*IssueTokenMsg)(nil)

const (
	pathIssueTokenMsg    = "nft/username/issue"
	pathAddAddressMsg    = "nft/username/address/add"
	pathRemoveAddressMsg = "nft/username/address/remove"
)

var (
	isValidID = regexp.MustCompile(`^[a-z0-9\.,\+\-_@]{4,64}$`).MatchString
)

// Path fulfills weave.Msg interface to allow routing
func (IssueTokenMsg) Path() string {
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

func (t *IssueTokenMsg) Validate() error {
	if err := validateID(t); err != nil {
		return err
	}
	if t == nil {
		return errors.ErrInternal("must not be nil")
	}
	if containsDuplicateChains(t.Addresses) {
		return nft.ErrDuplicateEntry()
	}
	if err := weave.Address(t.Owner).Validate(); err != nil {
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
