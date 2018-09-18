package nft

import (
	"time"

	"errors"

	"github.com/iov-one/weave"
)

// This could also be an array if we keep using int32 enums, convenient, eh?
type ActionApprovalsWrapper []*ActionApprovals
type Approvals map[ActionKind]ApprovalsMeta
type ApprovalsMeta []*Approval

func (a Approvals) Clone() Approvals {
	if a == nil {
		return nil
	}
	o := make(Approvals)
	for i, v := range a {
		o[i] = v.Clone()
	}
	return o
}
func (a Approvals) ByAction(action ActionKind) Approvals {
	res := make(map[ActionKind]ApprovalsMeta, 0)
	res[action] = a[action]
	return res
}

func (a Approvals) Add(action ActionKind, approval *Approval) Approvals {
	if _, ok := a[action]; !ok {
		a[action] = make([]*Approval, 0)
	}

	a[action] = append(a[action], approval)
	return a
}

func (a Approvals) ByAddress(to weave.Address) Approvals {
	res := make(map[ActionKind]ApprovalsMeta, 0)
	for k, v := range a {
		r := make([]*Approval, 0)
		for _, vv := range v {
			if vv.ToAccountAddress().Equals(to) {
				r = append(r, vv)
			}
		}
		if len(r) > 0 {
			res[k] = r
		}
	}
	return res
}

// TODO: Maybe we don't need to accept a list here anymore
func (a Approvals) Without(obsolete ...Approvals) Approvals {
	obsoleteMap := make(map[ActionKind]ApprovalsMeta, 0)
	res := make(map[ActionKind]ApprovalsMeta, 0)

	for _, o := range obsolete {
		for action, values := range o {
			if _, ok := obsoleteMap[action]; !ok {
				obsoleteMap[action] = make([]*Approval, 0)
			}
			obsoleteMap[action] = append(obsoleteMap[action], values...)
		}
	}

	//TODO: This can be further refactored into smaller pieces utilising more type wrappers
	for action, approvals := range a {
		obsoleteApprovals := obsoleteMap[action]
		for _, app := range approvals {
			found := false
			for _, obsoleteApproval := range obsoleteApprovals {
				if app.Equals(obsoleteApproval) {
					found = true
					break
				}
			}
			if !found {
				if _, ok := res[action]; !ok {
					res[action] = make([]*Approval, 0)
				}
				res[action] = append(res[action], app)
			}
		}
	}
	return res
}

//TODO: figure out the use-cases of this, we might also need to dereference Approvals, easier to clone?
func (a Approvals) AsValues() []ActionApprovals {
	r := make([]ActionApprovals, 0)
	for k, v := range a {
		r = append(r, ActionApprovals{k, v})
	}
	return r
}

func (a Approvals) AsOriginal() []*ActionApprovals {
	r := make([]*ActionApprovals, 0)
	for k, v := range a {
		r = append(r, &ActionApprovals{k, v})
	}
	return r
}

func (a Approvals) WithoutExpired() Approvals {
	res := make(map[ActionKind]ApprovalsMeta, 0)
	for action, approvals := range a {
		for _, app := range approvals {
			if app.Options.Timeout != 0 && time.Unix(0, app.Options.Timeout).Before(time.Now()) {
				continue
			}
			if app.Options.Count == 0 {
				continue
			}
			if _, ok := res[action]; !ok {
				res[action] = make([]*Approval, 0)
			}
			res[action] = append(res[action], app)
		}
	}
	return res
}

func (a Approvals) Exists() bool {
	return len(a) != 0
}

type ApprovalOperations struct {
	nft       Owned
	approvals *[]*ActionApprovals
}

func NewApprovalOperations(nft Owned, approvals *[]*ActionApprovals) *ApprovalOperations {
	return &ApprovalOperations{nft: nft, approvals: approvals}
}
func (o ApprovalOperations) List() Approvals {
	res := make(map[ActionKind]ApprovalsMeta, 0)
	for _, v := range *o.approvals {
		res[v.Action] = v.Approvals
	}
	return res
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
		for _, v := range a {
			if v.Options.Immutable {
				return errors.New("immutable and can not be changed")
			}
		}

	}
	*o.approvals = o.List().Without(approvalsToRemove).AsOriginal()
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

	*o.approvals = o.List().Add(action, &Approval{
		ToAccount: to, // todo: clone?
		Options:   op, // todo: Clone options
	}).AsOriginal()

	return nil
}
