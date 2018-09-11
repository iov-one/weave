package nft

import (
	"errors"
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
	"time"
)

const UnlimitedCount = -1

var _ orm.CloneableData = (*NonFungibleToken)(nil)

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
func (a Approvals) ByAction(action ActionlKind) Approvals {
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
func (a Approvals) Remove(obsoletes ...*Approval) Approvals {
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
	return a.Action == o.Action &&
		a.ToAccountAddress().Equals(o.ToAccountAddress()) &&
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
		time.Now().Before(time.Unix(0, a.Options.Timeout)) && a.Options.Count != 0
}

func (a *ApprovalOptions) Equals(o *ApprovalOptions) bool {
	if a == nil && o == nil || a == o {
		return true
	}
	return a.Immutilbe == o.Immutilbe && a.Count == o.Count && a.Timeout == o.Timeout
}

func (n *NonFungibleToken) Validate() error {
	// todo: impl
	return n.OwnerAddress().Validate()
}

func (n *NonFungibleToken) Copy() orm.CloneableData {
	// todo: impl
	return &NonFungibleToken{
		Owner:    n.Owner,
		Approval: Approvals(n.Approval).Clone(),
		Payload:  ClonePayload(n.Payload),
	}
}

func (n NonFungibleToken) OwnerAddress() weave.Address {
	return weave.Address(n.GetOwner())
}

func (n NonFungibleToken) Approvals(action ActionlKind) []Approval {
	return Approvals(n.Approval).ByAction(action).AsValues()
}
func (n *NonFungibleToken) SetApproval(action ActionlKind, to weave.Address, o *ApprovalOptions) error {
	if to == nil || to.Equals(n.Owner) {
		return errors.New("invalid destination account") // todo: move to errors
	}
	if o == nil {
		o = &ApprovalOptions{Count: UnlimitedCount}
	}
	// todo checks
	// todo: implement remove if exists aka map funktionality
	n.Approval = append(n.Approval, &Approval{
		Action:    action,
		ToAccount: to, // todo: clone?
		Options:   o,  // todo: Clone options
	})
	return nil
}
func (n *NonFungibleToken) RevokeApproval(action ActionlKind, to weave.Address) error {
	if to == nil || to.Equals(n.Owner) {
		return errors.New("invalid destination account") // todo: move to errors
	}
	approvalsToRemove := Approvals(n.Approval).ByAction(action).ByAddress(to)
	if len(approvalsToRemove) == 0 {
		return errors.New("does not exist")
	}
	for _, a := range approvalsToRemove {
		if a.Options.Immutilbe {
			return errors.New("immutible and can not be changed")
		}
	}

	n.Approval = Approvals(n.Approval).Remove(approvalsToRemove...)
	return nil
}

func (n *NonFungibleToken) Transfer(newOwner weave.Address) error {
	if newOwner == nil || newOwner.Equals(n.OwnerAddress()) {
		return errors.New("invalid destination account") // todo: move to errors
	}
	a := Approvals(n.Approval).ByAddress(newOwner).ByAction(ActionlKind_transferApproval).WithoutExpired()
	if len(a) == 0 {
		return errors.New("unauthorized")
	}
	n.Owner = []byte(newOwner) // todo: clone address?
	n.clearApprovals()
	return nil
}

func (n *NonFungibleToken) clearApprovals() {
	newApproval := make([]*Approval, 0, len(n.Approval))
	for _, a := range Approvals(n.Approval).WithoutExpired() {
		if !a.Options.Immutilbe {
			continue
		}
		newApproval = append(newApproval, a)
	}
	n.Approval = newApproval
}

func (n *NonFungibleToken) TakeAction(actor weave.Address, action ActionlKind, newPayload Payload) error {
	if actor == nil {
		return errors.New("invalid actor account") // todo: move to errors
	}
	// is allowed
	if !n.OwnerAddress().Equals(actor) {
		a := Approvals(n.Approval).ByAddress(actor).ByAction(action).WithoutExpired()
		if len(a) == 0 || !a[0].IsApplicable(actor) {
			return errors.New("unauthorized")
		}
		if a[0].Options.Count > 0 {
			a[0].Options.Count--
		}
	}

	// do action
	switch action {
	case ActionlKind_usageApproval: // do nothing
	case ActionlKind_updatePayloadApproval:
		n.Payload = newPayload
	default:
		return errors.New("unsupported action")
	}
	return nil
}

func ClonePayload(payload isNonFungibleToken_Payload) isNonFungibleToken_Payload {
	if payload == nil {
		return nil
	}
	// todo: implement proper
	return payload
}

func NewNonFungibleToken(key []byte, owner weave.Address) orm.Object {
	token := NonFungibleToken{
		Id:    key,
		Owner: owner,
	}
	return orm.NewSimpleObj(key, &token)
}

type BaseBucket interface {
	Issue(id []byte, owner weave.Address, initialPayload Payload) BaseNFT
	Load(id []byte) BaseNFT
	Revoke(id []byte)
}

type PersistentBaseBucket struct {
	orm.Bucket
}
type Payload isNonFungibleToken_Payload

// Note: we need to pass authorization info somehow,
// eg. via context or passed in explicitly
type BaseNFT interface {
	// read
	GetId() []byte
	OwnerAddress() weave.Address

	// permissions
	Approvals(action ActionlKind) []Approval
	SetApproval(action ActionlKind, to weave.Address, o *ApprovalOptions) error
	RevokeApproval(action ActionlKind, to weave.Address) error

	// usage: params depend on action type
	TakeAction(actor weave.Address, action ActionlKind, params Payload) error
	Transfer(newOwner weave.Address) error
}

//type Approval struct {
//	Account weave.Address
//	ApprovalOptions
//}
//
//type ApprovalOptions struct {
//	Timeout int
//	Count int
//	Immutible bool
//}
