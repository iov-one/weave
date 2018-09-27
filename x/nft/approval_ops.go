package nft

import (
	"errors"

	"github.com/iov-one/weave"
)

type ApprovalOps struct {
	owner weave.Address
	//TODO: Possibly define a type for it, e.g. *ActionApprovalsSet
	approvals *[]*ActionApprovals
}

//TODO: Sort errors and their codes
func NewApprovalOps(owner weave.Address, approvals *[]*ActionApprovals) *ApprovalOps {
	return &ApprovalOps{owner: owner, approvals: approvals}
}

func (o *ApprovalOps) List() Approvals {
	res := make(map[string]ApprovalMeta, 0)
	for _, v := range *o.approvals {
		res[v.Action] = v.Approvals
	}
	return res
}

func (o *ApprovalOps) Revoke(action string, from weave.Address) error {
	if from == nil || from.Equals(o.owner) {
		return errors.New("invalid account")
	}
	approvalsToRemove := o.List().ForAction(action).ForAddress(from)
	if len(approvalsToRemove) == 0 {
		return errors.New("does not exist")
	}
	for _, a := range approvalsToRemove {
		for _, v := range a {
			if v.Options.Immutable {
				return errors.New("immutable and can not be changed")
			}
		}
	}
	*o.approvals = o.List().Filter(approvalsToRemove).AsPersistable()
	return nil
}

//TODO: Figure out whether we need wildcard approvals, might be wise to add an ApprovalOptions flag
func (o *ApprovalOps) Grant(action string, to weave.Address, op ApprovalOptions, blockHeight int64, actions ...string) error {
	if to == nil || to.Equals(o.owner) {
		return errors.New("invalid destination account")
	}
	if !o.List().ForAddress(to).ForAction(action).FilterExpired(blockHeight).IsEmpty() {
		return errors.New("already exists")
	}

	approvals := o.List().Add(action, &Approval{
		Address: to,
		Options: op,
	})

	err := approvals.Validate(actions...)
	if err != nil {
		return err
	}

	*o.approvals = approvals.AsPersistable()
	return nil
}
