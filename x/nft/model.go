package nft

import (
	"errors"
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
)

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
func (a *Approval) Clone() *Approval {
	// todo: revisit to impl proper cloning
	x := *a
	return &x
}
func (a Approval) ToAccountAddress() weave.Address {
	if a.ToAccount == nil {
		return nil
	}
	return weave.Address(a.ToAccount)
}

func (n *NonFungibleToken) Validate() error {
	// todo: impl
	return nil
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
	r := make([]Approval, 0)
	for _, v := range n.Approval {
		if v.Action == action {
			r = append(r, *v)
		}
	}
	return r
}
func (n *NonFungibleToken) SetApproval(action ActionlKind, to weave.Address, o *ApprovalOptions) error {
	if to == nil || to.Equals(n.Owner) {
		return errors.New("invalid destination account") // todo: move to errors
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
	newApproval := make([]*Approval, 0, len(n.Approval))

	for _, a := range n.Approval {
		if to.Equals(a.ToAccountAddress()) && a.Action == action {
			continue
		}
		newApproval = append(newApproval, a)
	}
	n.Approval = newApproval
	return nil
}

func (n *NonFungibleToken) Transfer(newOwner weave.Address) error {
	if newOwner == nil {
		return errors.New("invalid destination account") // todo: move to errors
	}
	approval := n.Approvals(ActionlKind_transferApproval)
	found := false
	for _, a := range approval {
		if newOwner.Equals(a.ToAccountAddress()) { // && todo not expired
			found = true
			break
		}
	}
	if !found {
		return errors.New("unauthorized")
	}
	n.Owner = []byte(newOwner) // todo: clone address?
	n.clearApprovals()
	return nil
}

func (n *NonFungibleToken) clearApprovals() {
	newApproval := make([]*Approval, 0, len(n.Approval))
	for _, a := range n.Approval {
		if a.Options != nil && a.Options.Immutilbe { // && todo not expired
			// todo: also skip all where the new owner makes no sense
			newApproval = append(newApproval, a)
		}
	}
	n.Approval = newApproval
}

func ClonePayload(payload isNonFungibleToken_Payload) isNonFungibleToken_Payload {
	if payload == nil {
		return nil
	}
	// todo: implement proper
	return payload
}

func NewNonFungibleToken(key []byte) orm.Object {
	token := NonFungibleToken{Id: key}
	return orm.NewSimpleObj(key, &token)
}

type BaseBucket interface {
	Issue(id []byte, owner weave.Address, initialPayload nftPayload) BaseNFT
	Load(id []byte) BaseNFT
	Revoke(id []byte)
}

type PersistentBaseBucket struct {
	orm.Bucket
}
type nftPayload isNonFungibleToken_Payload

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
	//TakeAction(actor weave.Address, action string, params interface{})
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
