package nft

import (
	"errors"

	"github.com/iov-one/weave"
)

type ApprovalOps struct {
	nft Owned
	//TODO: Possibly define a type for it, e.g. *ActionApprovalsSet
	approvals *[]*ActionApprovals
}

//TODO: Sort errors and their codes
func NewApprovalOps(nft Owned, approvals *[]*ActionApprovals) *ApprovalOps {
	return &ApprovalOps{nft: nft, approvals: approvals}
}

func (o *ApprovalOps) List() Approvals {
	res := make(map[string]ApprovalMeta, 0)
	for _, v := range *o.approvals {
		res[v.Action] = v.Approvals
	}
	return res
}

func (o *ApprovalOps) Revoke(action string, from weave.Address) error {
	if from == nil || from.Equals(o.nft.OwnerAddress()) {
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
	*o.approvals = o.List().Filter(approvalsToRemove).AsOriginal()
	return nil
}

func (o *ApprovalOps) Grant(action string, to weave.Address, op *ApprovalOptions, blockHeight int64) error {
	if to == nil || to.Equals(o.nft.OwnerAddress()) {
		return errors.New("invalid destination account")
	}
	if !o.List().ForAddress(to).ForAction(action).FilterExpired(blockHeight).IsEmpty() {
		return errors.New("already exists")
	}
	if op == nil {
		op = &ApprovalOptions{Count: UnlimitedCount}
	}
	approvals := o.List().Add(action, &Approval{
		Address: to,
		Options:   op.Clone(),
	})

	err := approvals.Validate()
	if err != nil {
		return err
	}

	*o.approvals = approvals.AsOriginal()
	return nil
}