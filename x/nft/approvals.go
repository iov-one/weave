package nft

import (
	"time"

	"errors"

	"github.com/iov-one/weave"
)

type Approvals []*Approval

func (a Approvals) Clone() Approvals {
	if a == nil {
		return nil
	}
	o := make([]*Approval, len(a))
	for i, v := range a {
		o[i] = v.Clone()
	}
	return o
}
func (a Approvals) ByAction(action ActionKind) Approvals {
	r := make([]*Approval, 0)
	for _, v := range a {
		if v.Action == action {
			r = append(r, v)
		}
	}
	return r
}
func (a Approvals) ByAddress(to weave.Address) Approvals {
	r := make([]*Approval, 0)
	for _, v := range a {
		if v.ToAccountAddress().Equals(to) {
			r = append(r, v)
		}
	}
	return r
}
func (a Approvals) Without(obsoletes ...*Approval) Approvals {
	r := make([]*Approval, 0)
	for _, v := range a {
		found := false
		for _, o := range obsoletes { // todo: revisit
			if v.Equals(o) {
				found = true
				break
			}
		}
		if !found {
			r = append(r, v)
		}
	}
	return r
}

func (a Approvals) AsValues() []Approval {
	r := make([]Approval, 0)
	for _, v := range a {
		r = append(r, *v)
	}
	return r
}

func (a Approvals) WithoutExpired() Approvals {
	r := make([]*Approval, 0)
	for _, v := range a {
		if v.Options.Timeout != 0 && time.Unix(0, v.Options.Timeout).Before(time.Now()) {
			continue
		}
		if v.Options.Count == 0 {
			continue
		}
		r = append(r, v)
	}
	return r
}

func (a Approvals) Exists() bool {
	return len(a) != 0
}

type ApprovalOperations struct {
	nft       Owned
	approvals *[]*Approval
}

func NewApprovalOperations(nft Owned, approvals *[]*Approval) *ApprovalOperations {
	return &ApprovalOperations{nft: nft, approvals: approvals}
}
func (o ApprovalOperations) List() Approvals {
	return *o.approvals
}
func (o *ApprovalOperations) Revoke(action ActionKind, to weave.Address) error {
	if to == nil || to.Equals(o.nft.OwnerAddress()) {
		return errors.New("invalid destination account") // todo: move to errors
	}

	approvalsToRemove := o.List().ByAction(action).ByAddress(to)
	if len(approvalsToRemove) == 0 {
		return errors.New("does not exist")
	}
	for _, a := range approvalsToRemove {
		if a.Options.Immutilbe {
			return errors.New("immutible and can not be changed")
		}
	}
	*o.approvals = o.List().Without(approvalsToRemove...)
	return nil
}

func (o *ApprovalOperations) Set(action ActionKind, to weave.Address, op *ApprovalOptions) error {
	if to == nil || to.Equals(o.nft.OwnerAddress()) {
		return errors.New("invalid destination account") // todo: move to errors
	}
	if o.List().ByAddress(to).ByAction(action).WithoutExpired().Exists() {
		return errors.New("already exists") // todo: move to erorrs
	}
	if op == nil {
		op = &ApprovalOptions{Count: UnlimitedCount}
	}

	// todo: find and replace
	*o.approvals = append(*o.approvals, &Approval{
		Action:    action,
		ToAccount: to, // todo: clone?
		Options:   op, // todo: Clone options
	})

	return nil
}
