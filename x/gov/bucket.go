package gov

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
)

// ElectorateBucket is the persistent bucket for Electorate object.
type ElectorateBucket struct {
	orm.Bucket
	idSeq orm.Sequence
}

// NewRevenueBucket returns a bucket for managing electorate.
func NewElectorateBucket() *ElectorateBucket {
	b := orm.NewBucket("electorate", orm.NewSimpleObj(nil, &Electorate{}))
	return &ElectorateBucket{
		Bucket: b,
		idSeq:  b.Sequence("id"),
	}
}

// Build assigns an ID to given electorate instance and returns it as an orm
// Object. It does not persist the object in the store.
func (b *ElectorateBucket) Build(db weave.KVStore, e *Electorate) orm.Object {
	key := b.idSeq.NextVal(db)
	return orm.NewSimpleObj(key, e)
}

// GetElectorate loads the electorate for the given id. If it does not exist then ErrNotFound is returned.
func (b *ElectorateBucket) GetElectorate(db weave.KVStore, id []byte) (*Electorate, error) {
	obj, err := b.Get(db, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load electorate")
	}
	if obj == nil || obj.Value() == nil {
		return nil, errors.Wrap(errors.ErrNotFound, "unknown id")
	}
	rev, ok := obj.Value().(*Electorate)
	if !ok {
		return nil, errors.Wrapf(errors.ErrInvalidModel, "invalid type: %T", obj.Value())
	}
	return rev, nil
}

// NewElectionRulesBucket is the persistent bucket for ElectionRules .
type ElectionRulesBucket struct {
	orm.Bucket
	idSeq orm.Sequence
}

// NewElectionRulesBucket returns a bucket for managing election rules.
func NewElectionRulesBucket() *ElectionRulesBucket {
	b := orm.NewBucket("electnrule", orm.NewSimpleObj(nil, &ElectionRule{}))
	return &ElectionRulesBucket{
		Bucket: b,
		idSeq:  b.Sequence("id"),
	}
}

// Build assigns an ID to given election rule instance and returns it as an orm
// Object. It does not persist the object in the store.
func (b *ElectionRulesBucket) Build(db weave.KVStore, r *ElectionRule) orm.Object {
	key := b.idSeq.NextVal(db)
	return orm.NewSimpleObj(key, r)
}

// GetElectionRule loads the electorate for the given id. If it does not exist then ErrNotFound is returned.
func (b *ElectionRulesBucket) GetElectionRule(db weave.KVStore, id []byte) (*ElectionRule, error) {
	obj, err := b.Get(db, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load election rule")
	}
	if obj == nil || obj.Value() == nil {
		return nil, errors.Wrap(errors.ErrNotFound, "unknown id")
	}
	rev, ok := obj.Value().(*ElectionRule)
	if !ok {
		return nil, errors.Wrapf(errors.ErrInvalidModel, "invalid type: %T", obj.Value())
	}
	return rev, nil
}

// ProposalBucket is the persistent bucket for governance proposal objects.
type ProposalBucket struct {
	orm.Bucket
	idSeq orm.Sequence
}

// NewProposalBucket returns a bucket for managing electorate.
func NewProposalBucket() *ProposalBucket {
	b := orm.NewBucket("proposal", orm.NewSimpleObj(nil, &TextProposal{}))
	return &ProposalBucket{
		Bucket: b,
		idSeq:  b.Sequence("id"),
	}
}

// Build assigns an ID to given proposal instance and returns it as an orm
// Object. It does not persist the object in the store.
func (b *ProposalBucket) Build(db weave.KVStore, e *TextProposal) orm.Object {
	key := b.idSeq.NextVal(db)
	return orm.NewSimpleObj(key, e)
}

// GetTextProposal loads the proposal for the given id. If it does not exist then ErrNotFound is returned.
func (b *ProposalBucket) GetTextProposal(db weave.KVStore, id []byte) (*TextProposal, error) {
	obj, err := b.Get(db, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load proposal")
	}
	if obj == nil || obj.Value() == nil {
		return nil, errors.Wrap(errors.ErrNotFound, "unknown id")
	}
	rev, ok := obj.Value().(*TextProposal)
	if !ok {
		return nil, errors.Wrapf(errors.ErrInvalidModel, "invalid type: %T", obj.Value())
	}
	return rev, nil
}

// Update stores the given proposal and id in the persistence store.
func (b *ProposalBucket) Update(db weave.KVStore, id []byte, obj *TextProposal) error {
	if err := b.Save(db, orm.NewSimpleObj(id, obj)); err != nil {
		return errors.Wrap(err, "failed to save")
	}
	return nil
}

// VoteBucket is the persistence bucket.
type VoteBucket struct {
	orm.Bucket
}

// NewProposalBucket returns a bucket for managing electorate.
func NewVoteBucket() *VoteBucket {
	b := orm.NewBucket("vote", orm.NewSimpleObj(nil, &Vote{}))
	return &VoteBucket{
		Bucket: b,
	}
}

func (b *VoteBucket) Build(db weave.KVStore, proposalID []byte, vote Vote) orm.Object {
	compositeKey := compositeKey(proposalID, vote.Elector.Signature)
	return orm.NewSimpleObj(compositeKey, &vote)
}

func compositeKey(proposalID []byte, address weave.Address) []byte {
	return append(proposalID, address...)
}

func (b *VoteBucket) HasVoted(db weave.KVStore, proposalID []byte, elector weave.Address) (bool, error) {
	obj, err := b.Get(db, compositeKey(proposalID, elector))
	if err != nil {
		return false, errors.Wrap(err, "failed to load vote")
	}
	return obj != nil && obj.Value() != nil, nil
}
