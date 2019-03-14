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

// Path returns the routing path for this message
func (*IssueTokenMsg) Path() string {
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

func (m *IssueTokenMsg) Validate() error {
	if err := validateID(m.ID); err != nil {
		return err
	}
	if err := m.Details.Validate(); err != nil {
		return err
	}

	addr := weave.Address(m.Owner)

	if err := addr.Validate(); err != nil {
		return err
	}
	//TODO: This is being validated on model save
	//so in our case both check and deliver - double
	//work?
	if err := nft.NewApprovalOps(addr, &m.Approvals).
		List().
		Validate(); err != nil {
		return err
	}

	return nil
}

func (m *AddChainAddressMsg) Validate() error {
	if err := validateID(m.UsernameID); err != nil {
		return err
	}
	address := ChainAddress{BlockchainID: m.GetBlockchainID(), Address: m.GetAddress()}
	return address.Validate()
}

func (m *RemoveChainAddressMsg) Validate() error {
	if err := validateID(m.UsernameID); err != nil {
		return err
	}
	address := ChainAddress{BlockchainID: m.GetBlockchainID(), Address: m.GetAddress()}
	return address.Validate()
}

func validateID(id []byte) error {
	if id == nil {
		return errors.Wrap(errors.ErrInvalidInput, "must not be nil")
	}
	if !isValidID(string(id)) {
		return errors.Wrapf(errors.ErrInvalidInput, "id: %s", nft.PrintableID(id))
	}
	return nil
}
