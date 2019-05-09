package nft

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &NonFungibleToken{}, migration.NoModification)
}

var _ orm.CloneableData = (*NonFungibleToken)(nil)

func (m *NonFungibleToken) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if !isValidTokenID(m.ID) {
		return errors.Wrapf(errors.ErrInput, "id: %s", PrintableID(m.ID))
	}

	if err := m.Owner.Validate(); err != nil {
		return err
	}

	return nil
}

func (m *NonFungibleToken) Copy() orm.CloneableData {
	return m.Clone()
}

func (m *NonFungibleToken) Clone() *NonFungibleToken {
	actionApprovals := make([]ActionApprovals, len(m.ActionApprovals))
	for i, v := range m.ActionApprovals {
		actionApprovals[i] = v.Clone()
	}
	return &NonFungibleToken{
		Metadata:        m.Metadata.Copy(),
		ID:              m.ID,
		Owner:           m.Owner,
		ActionApprovals: actionApprovals,
	}
}

func NewNonFungibleToken(key []byte, owner weave.Address, approvals []ActionApprovals) *NonFungibleToken {
	return &NonFungibleToken{
		Metadata:        &weave.Metadata{Schema: 1},
		ID:              key,
		Owner:           owner,
		ActionApprovals: approvals,
	}
}

func (u *NonFungibleToken) OwnerAddress() weave.Address {
	return weave.Address(u.Owner)
}

func (m *NonFungibleToken) Approvals() *ApprovalOps {
	return NewApprovalOps(m.Owner, &m.ActionApprovals)
}

func (m *NonFungibleToken) SetApprovals(a Approvals) {
	m.ActionApprovals = a.AsPersistable()
}

func (m *NonFungibleToken) HasApproval(actor weave.Address, action Action) bool {
	return !NewApprovalOps(m.Owner, &m.ActionApprovals).
		List().ForAction(action).ForAddress(actor).IsEmpty()
}

type BaseNFT interface {
	Owned
	//GetId() []byte
	Approvals() *ApprovalOps
	//Set new approvals
	SetApprovals(Approvals)
}

//TODO: Better name
type Identified interface {
	GetID() []byte
}
