package nft

import (
	"github.com/iov-one/weave"
)

var _ weave.Msg = (*IssueTokenMsg)(nil)

const pathIssue = "nft/issue"

// Path returns the routing path for this message
func (*IssueTokenMsg) Path() string {
	return pathIssue
}
func (t *IssueTokenMsg) Validate() error {
	// Todo: impl
	return nil
}
