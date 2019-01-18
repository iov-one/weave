package bootstrap_node

import (
	"regexp"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x/nft"
)

var _ weave.Msg = (*IssueTokenMsg)(nil)

const (
	pathIssueTokenMsg = "nft/bootstrap_node/issue"
)

var (
	//todo: revisit pattern
	IsValidID = regexp.MustCompile(`^[A-Za-z0-9]{3,20}$`).MatchString
)

// Path returns the routing path for this message
func (*IssueTokenMsg) Path() string {
	return pathIssueTokenMsg
}

func (i *IssueTokenMsg) Validate() error {
	if i == nil {
		return errors.ErrInternal("must not be nil")
	}
	owner := weave.Address(i.Owner)
	if err := owner.Validate(); err != nil {
		return err
	}

	if !IsValidID(string(i.ID)) {
		return nft.ErrInvalidID(i.ID)
	}
	if err := i.Details.Validate(); err != nil {
		return err
	}

	//TODO: This is being validated on model save
	//so in our case both check and deliver - double
	//work?
	if err := nft.NewApprovalOps(owner, &i.Approvals).
		List().
		Validate(); err != nil {
		return err
	}

	return nil
}
