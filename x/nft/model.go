package nft

import (
	"errors"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
)

const UnlimitedCount = -1

var _ orm.CloneableData = (*NonFungibleToken)(nil)

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
		(a.Options.Timeout == 0 || time.Now().Before(time.Unix(0, a.Options.Timeout))) &&
		a.Options.Count != 0
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
		Owner:     n.Owner,
		Approvals: Approvals(n.Approvals).Clone(),
		Details:   n.Details.Clone(),
	}
}

func (n NonFungibleToken) OwnerAddress() weave.Address {
	return weave.Address(n.GetOwner())
}

func (n NonFungibleToken) XApprovals(action ActionKind) []Approval {
	return Approvals(n.Approvals).ByAction(action).WithoutExpired().AsValues()
}

func (n *NonFungibleToken) SetApproval(action ActionKind, to weave.Address, o *ApprovalOptions) error {
	if to == nil || to.Equals(n.Owner) {
		return errors.New("invalid destination account") // todo: move to errors
	}
	if Approvals(n.Approvals).ByAddress(to).ByAction(action).WithoutExpired().Exists() {
		return errors.New("already exists") // todo: move to erorrs
	}

	if o == nil {
		o = &ApprovalOptions{Count: UnlimitedCount}
	}

	// todo: implement remove if exists aka map funktionality
	n.Approvals = append(n.Approvals, &Approval{
		Action:    action,
		ToAccount: to, // todo: clone?
		Options:   o,  // todo: Clone options
	})
	return nil
}

func (n *NonFungibleToken) RevokeApproval(action ActionKind, to weave.Address) error {
	if to == nil || to.Equals(n.Owner) {
		return errors.New("invalid destination account") // todo: move to errors
	}
	approvalsToRemove := Approvals(n.Approvals).ByAction(action).ByAddress(to)
	if len(approvalsToRemove) == 0 {
		return errors.New("does not exist")
	}
	for _, a := range approvalsToRemove {
		if a.Options.Immutilbe {
			return errors.New("immutible and can not be changed")
		}
	}

	n.Approvals = Approvals(n.Approvals).Remove(approvalsToRemove...)
	return nil
}

func (n *NonFungibleToken) Transfer(newOwner weave.Address) error {
	if newOwner == nil || newOwner.Equals(n.OwnerAddress()) {
		return errors.New("invalid destination account") // todo: move to errors
	}
	// todo: revisit checks
	approvals := Approvals(n.Approvals).ByAddress(newOwner).
		ByAction(ActionKind_transferApproval).WithoutExpired()
	if !approvals.Exists() || !approvals[0].IsApplicable(newOwner) {
		return errors.New("unauthorized") // todo: move to errors
	}
	n.Owner = []byte(newOwner) // todo: clone address?
	n.clearApprovals()
	return nil
}

func (n *NonFungibleToken) clearApprovals() {
	newApproval := make([]*Approval, 0, len(n.Approvals))
	for _, a := range Approvals(n.Approvals).WithoutExpired() {
		if !a.Options.Immutilbe {
			continue
		}
		newApproval = append(newApproval, a)
	}
	n.Approvals = newApproval
}

func (n *NonFungibleToken) TakeAction(actor weave.Address, action ActionKind, newDetails Payload) error {
	if actor == nil {
		return errors.New("invalid actor account") // todo: move to errors
	}
	// is allowed
	if !n.OwnerAddress().Equals(actor) {
		a := Approvals(n.Approvals).ByAddress(actor).ByAction(action).WithoutExpired()
		if len(a) == 0 || !a[0].IsApplicable(actor) {
			return errors.New("unauthorized")
		}
		if a[0].Options.Count > 0 {
			a[0].Options.Count--
		}
	}

	// do action
	switch action {
	case ActionKind_usageApproval: // do nothing
	case ActionKind_updatePayloadApproval:
		// todo: check type so that we do not update the wrong kind
		n.Details = &TokenDetails{Payload: newDetails}
	default:
		return errors.New("unsupported action")
	}
	return nil
}

func (d *TokenDetails) Clone() *TokenDetails {
	if d == nil {
		return nil
	}
	// todo: implement proper
	return d
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
type Payload isTokenDetails_Payload

// Note: we need to pass authorization info somehow,
// eg. via context or passed in explicitly
type BaseNFT interface {
	// read
	GetId() []byte
	OwnerAddress() weave.Address

	// permissions
	XApprovals(action ActionKind) []Approval // todo: come up with a better name
	SetApproval(action ActionKind, to weave.Address, o *ApprovalOptions) error
	RevokeApproval(action ActionKind, to weave.Address) error

	// usage: params depend on action type
	TakeAction(actor weave.Address, action ActionKind, params Payload) error
	Transfer(newOwner weave.Address) error
}
