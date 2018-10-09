package nft

import (
	"fmt"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

const (
	ActionUpdateDetails = "BASE_UPDATE_DETAILS"
	ActionTransfer      = "BASE_ACTION_TRANSFER"
)

const UnlimitedCount = -1

type ApprovalMeta []*Approval
type Approvals map[string]ApprovalMeta

func (m *ActionApprovals) Clone() *ActionApprovals {
	x := *m
	approvals := make([]*Approval, len(m.Approvals))
	for i, v := range m.Approvals {
		approvals[i] = v.Clone()
	}
	return &x
}

func (m *Approval) Clone() *Approval {
	x := *m
	return &x
}
func (m ApprovalMeta) Clone() ApprovalMeta {
	x := m
	approvals := make([]*Approval, 0)
	for k, v := range x {
		approvals[k] = v.Clone()
	}
	return approvals
}

func (a Approval) AsAddress() weave.Address {
	if a.Address == nil {
		return nil
	}
	return weave.Address(a.Address)
}

func (a *Approval) Equals(o *Approval) bool {
	if a == nil && o == nil || a == o {
		return true
	}
	return a.AsAddress().Equals(o.AsAddress()) &&
		a.Options.Equals(o.Options)
}

func (a ApprovalOptions) Equals(o ApprovalOptions) bool {
	return a.Immutable == o.Immutable && a.Count == o.Count && a.UntilBlockHeight == o.UntilBlockHeight
}

//TODO: decide what we need to validate here and what before
//This requires all the model-specific actions to be passed here
func (m Approvals) Validate(actions ...string) error {
	actions = append(actions, []string{ActionUpdateDetails, ActionTransfer}...)
	actionMap := make(map[string]struct{}, 0)
	for _, action := range actions {
		actionMap[action] = struct{}{}
	}

	for action := range m {
		if _, ok := actionMap[action]; !ok {
			return errors.ErrInternal(fmt.Sprintf("illegal action: %s", action))
		}
	}

	// todo: also validate that options are not nil, maybe best to do it at the moment of granting
	return nil
}

func (m Approvals) FilterExpired(blockHeight int64) Approvals {
	res := make(map[string]ApprovalMeta, 0)
	for action, approvals := range m {
		for _, approval := range approvals {
			if approval.Options.UntilBlockHeight < blockHeight {
				continue
			}
			if approval.Options.Count == 0 {
				continue
			}
			if _, ok := res[action]; !ok {
				res[action] = make([]*Approval, 0)
			}
			res[action] = append(res[action], approval)
		}
	}
	return res
}

func (m Approvals) AsPersistable() []*ActionApprovals {
	r := make([]*ActionApprovals, 0)
	for k, v := range m {
		r = append(r, &ActionApprovals{k, v})
	}
	return r
}

func (m Approvals) IsEmpty() bool {
	return len(m) > 0
}

func (m Approvals) MetaByAction(action string) ApprovalMeta {
	return m[action]
}

func (m Approvals) ForAction(action string) Approvals {
	res := make(map[string]ApprovalMeta, 0)
	res[action] = m.MetaByAction(action)
	return res
}

func (m Approvals) ForAddress(addr weave.Address) Approvals {
	res := make(map[string]ApprovalMeta, 0)
	for k, v := range m {
		r := make([]*Approval, 0)
		for _, vv := range v {
			if vv.AsAddress().Equals(addr) {
				r = append(r, vv)
			}
		}
		if len(r) > 0 {
			res[k] = r
		}
	}
	return res
}

func (m Approvals) Filter(obsolete Approvals) Approvals {
	res := make(map[string]ApprovalMeta, 0)

ApprovalsLoop:
	for action, approvals := range m {
		obsoleteApprovals := obsolete[action]
		for _, approval := range approvals {
			for _, obsoleteApproval := range obsoleteApprovals {
				if approval.Equals(obsoleteApproval) {
					continue ApprovalsLoop
				}
			}
			res[action] = append(res[action], approval)
		}
	}
	return res
}

func (m Approvals) Add(action string, approval *Approval) Approvals {
	m[action] = append(m[action], approval)
	return m
}
