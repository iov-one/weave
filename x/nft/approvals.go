package nft

import "github.com/iov-one/weave/errors"

const (
	ActionKindUpdateDetails = "baseUpdateDetails"
)

func (i *ActionApprovals) Validate() error {
	if i == nil {
		return errors.ErrInternal("must not be nil")
	}
	// todo: do proper validation
	return nil
}
