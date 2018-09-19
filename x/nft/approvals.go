package nft

import (
	"fmt"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave"
)

const (
	ActionUpdateDetails = "baseUpdateDetails"
	ActionTransfer = "baseActionTransfer"
)

const UnlimitedCount = -1

//TODO: Revisit this typename
type ApprovalMeta []*Approval
type Approvals map[string]ApprovalMeta

func(m *ActionApprovals) Clone() *ActionApprovals {
	x := *m
	approvals := make([]*Approval, 0)
	for k, v := range m.Approvals {
		approvals[k] = v.Clone()
	}
	return &x
}

func(m *Approval) Clone() *Approval {
	x := *m
	// We should not allow nil options here, so a panic is fine
	x.Options = x.Options.Clone()
	return &x
}

func(m *ApprovalOptions) Clone() *ApprovalOptions {
	x := *m
	return &x
}

func(m ApprovalMeta) Clone() ApprovalMeta {
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

func (a *ApprovalOptions) Equals(o *ApprovalOptions) bool {
	if a == nil && o == nil || a == o {
		return true
	}
	return a.Immutable == o.Immutable && a.Count == o.Count && a.UntilBlockHeight == o.UntilBlockHeight
}

//TODO: decide what we need to validate here and what before
func (m Approvals) Validate(actions ...string) error {
	actions = append(actions, []string{ActionUpdateDetails, ActionTransfer}...)
	actionMap := make(map[string]bool, 0)
	for _, action := range actions {
		actionMap[action] = true
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
		for _, app := range approvals {
			if app.Options.UntilBlockHeight < blockHeight {
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

func (m Approvals) AsOriginal() []*ActionApprovals {
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

//TODO: See if this can be further simplified
func (m Approvals) Filter(obsolete Approvals) Approvals {
	res := make(map[string]ApprovalMeta, 0)

	for action, approvals := range m {
		obsoleteApprovals := obsolete[action]
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

func (m Approvals) Add(action string, approval *Approval) Approvals {
	// TODO: Should we validate action here instead? would make sense.
	if _, ok := m[action]; !ok {
		m[action] = make([]*Approval, 0)
	}
	m[action] = append(m[action], approval)
	return m
}