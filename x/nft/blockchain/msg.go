package blockchain

import (
	"github.com/iov-one/weave"
)

const (
	pathIssue  = "nft/blockchain/issue"
	pathUpdate = "nft/blockchain/update"
)

var (
	_ weave.Msg = (*IssueTokenMsg)(nil)
	_ weave.Msg = (*UpdateTokenMsg)(nil)
)

// Path returns the routing path for this message
func (*IssueTokenMsg) Path() string {
	return pathIssue
}
func (t *IssueTokenMsg) Validate() error {
	// Todo: impl
	return nil
}

// Path returns the routing path for this message
func (*UpdateTokenMsg) Path() string {
	return pathUpdate
}
func (t *UpdateTokenMsg) Validate() error {
	// Todo: impl
	return nil
}
