package username

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x/nft"
	"regexp"
)

var _ weave.Msg = (*IssueTokenMsg)(nil)

const (
	pathIssueTokenMsg = "nft/username/issue"
)

var (
	isValidID = regexp.MustCompile(`^[a-zA-Z0-9_@\+\-\.]{4,256}$`).MatchString
)

// Path returns the routing path for this message
func (*IssueTokenMsg) Path() string {
	return pathIssueTokenMsg
}

func (i *IssueTokenMsg) Validate() error {
	if i == nil {
		return errors.ErrInternal("must not be nil")
	}
	if err := weave.Address(i.Owner).Validate(); err != nil {
		return err
	}

	if !isValidID(string(i.Id)) {
		return nft.ErrInvalidID()
	}
	if err := i.Details.Validate(); err != nil {
		return err
	}
	for _, a := range i.Approvals {
		if err := a.Validate(); err != nil {
			return err
		}
	}
	return nil
}
