package nft

import (
	"errors"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
)

const UnlimitedCount = -1

var _ orm.CloneableData = (*NonFungibleToken)(nil)

func (a ApprovalsMeta) Clone() ApprovalsMeta {
	x := a
	approvals := make([]*Approval, 0)
	for k, v := range x {
		approvals[k] = v.Clone()
	}
	return approvals
}

func (a ApprovalsMeta) IsApplicable(to weave.Address) bool {
	for _, v := range a {
		if v.IsApplicable(to) {
			return true
		}
	}
	return false
}

func (a *ActionApprovals) Clone() *ActionApprovals {
	x := *a
	approvals := make([]*Approval, 0)
	for k, v := range a.Approvals {
		approvals[k] = v.Clone()
	}
	return &x
}

func (a ActionApprovalsWrapper) Clone() ActionApprovalsWrapper {
	approvals := make([]*ActionApprovals, 0)
	for k, v := range a {
		approvals[k] = v.Clone()
	}
	return approvals
}

func (a *Approval) Clone() *Approval {
	// todo: revisit to impl proper cloning
	x := *a
	if x.Options == nil {
		x.Options = &ApprovalOptions{}
	}
	return &x
}
func (a *Approval) Equals(o *Approval) bool {
	if a == nil && o == nil || a == o {
		return true
	}
	return a.ToAccountAddress().Equals(o.ToAccountAddress()) &&
		a.Options.Equals(o.Options)
}

func (a Approval) ToAccountAddress() weave.Address {
	if a.ToAccount == nil {
		return nil
	}
	return weave.Address(a.ToAccount)
}
func (a Approval) IsApplicable(to weave.Address) bool {
	return to != nil && a.ToAccountAddress().Equals(to) &&
		(a.Options.Timeout == 0 || time.Now().Before(time.Unix(0, a.Options.Timeout))) &&
		a.Options.Count != 0
}

func (a *ApprovalOptions) Equals(o *ApprovalOptions) bool {
	if a == nil && o == nil || a == o {
		return true
	}
	return a.Immutable == o.Immutable && a.Count == o.Count && a.Timeout == o.Timeout
}

func (n *NonFungibleToken) Validate() error {
	// todo: impl
	return n.OwnerAddress().Validate()
}

func (n *NonFungibleToken) Copy() orm.CloneableData {
	// todo: impl
	return &NonFungibleToken{
		Owner:           n.Owner,
		ActionApprovals: ActionApprovalsWrapper(n.ActionApprovals).Clone(),
	}
}

func (n NonFungibleToken) OwnerAddress() weave.Address {
	return weave.Address(n.GetOwner())
}

func (n *NonFungibleToken) Transfer(newOwner weave.Address) error {
	if newOwner == nil || newOwner.Equals(n.OwnerAddress()) {
		return errors.New("invalid destination account") // todo: move to errors
	}
	// todo: revisit checks
	if !n.HasApproval(newOwner, ActionKind_Transfer) {
		return errors.New("not approved") // todo: move to errors
	}
	n.Owner = []byte(newOwner) // todo: clone address?
	n.clearApprovals()
	return nil
}

func (n *NonFungibleToken) clearApprovals() {
	ops := NewApprovalOperations(n, &n.ActionApprovals)
	n.ActionApprovals = ops.List().WithoutExpired().AsOriginal()
}

func (n *NonFungibleToken) HasApproval(to weave.Address, action ActionKind) bool {
	ops := NewApprovalOperations(n, &n.ActionApprovals)

	approvals := ops.List().ByAddress(to).
		ByAction(action).WithoutExpired()
	return approvals.Exists() && approvals[action][0].IsApplicable(to)
}

//func (n *NonFungibleToken) TakeAction(actor weave.Address, action ActionKind, newDetails Payload) error {
//	if actor == nil {
//		return errors.New("invalid actor account") // todo: move to errors
//	}
//	// is allowed
//	if !n.OwnerAddress().Equals(actor) {
//		a := Approvals(n.Approvals).ByAddress(actor).ByAction(action).WithoutExpired()
//		if len(a) == 0 || !a[0].IsApplicable(actor) {
//			return errors.New("unauthorized")
//		}
//		if a[0].Options.Count > 0 {
//			a[0].Options.Count--
//		}
//	}
//
//	// do action
//	switch action {
//	case ActionKind_Usage: // do nothing
//	default:
//		return errors.New("unsupported action")
//	}
//	return nil
//}

func NewNonFungibleToken(key []byte, owner weave.Address) *NonFungibleToken {
	return &NonFungibleToken{
		Id:    key,
		Owner: owner,
	}
}

//// Note: we need to pass authorization info somehow,
//// eg. via context or passed in explicitly
type BaseNFT interface {
	//	// read
	Owned
	//	GetId() []byte
	//
	//	// permissions
	Approvals() *ApprovalOperations
	//
	//	// usage: params depend on action type
	Transfer(newOwner weave.Address) error
}
