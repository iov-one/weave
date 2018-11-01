package approvals

import "github.com/iov-one/weave"

const (
	pathCreateApprovalMsg = "approvals/create"
	pathUpdateApprovalMsg = "approvals/update"

	creationCost int64 = 300 // 3x more expensive than SendMsg
	updateCost   int64 = 150 // Half the creation cost
)

// Path fulfills weave.Msg interface to allow routing
func (CreateApprovalMsg) Path() string {
	return pathCreateApprovalMsg
}

// Validate enforces sigs and threshold boundaries
func (c *CreateApprovalMsg) Validate() error {
	if err := weave.Address(c.Address).Validate(); err != nil {
		return err
	}
	return nil
}

// Path fulfills weave.Msg interface to allow routing
func (UpdateApprovalMsg) Path() string {
	return pathUpdateApprovalMsg
}

// Validate enforces sigs and threshold boundaries
func (c *UpdateApprovalMsg) Validate() error {
	if err := weave.Address(c.Address).Validate(); err != nil {
		return err
	}
	return nil
}
