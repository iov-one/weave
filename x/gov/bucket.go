package gov

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

// ElectorateBucket is the persistent bucket for Electorate object.
type ElectorateBucket struct {
	orm.VersioningBucket
}

// NewRevenueBucket returns a bucket for managing electorate.
func NewElectorateBucket() *ElectorateBucket {
	b := migration.NewBucket(packageName, "electorate", orm.NewSimpleObj(nil, &Electorate{}))
	return &ElectorateBucket{
		VersioningBucket: orm.WithVersioning(orm.WithSeqIDGenerator(b, "id")),
	}
}

func asElectorate(obj orm.Object) (*Electorate, error) {
	rev, ok := obj.Value().(*Electorate)
	if !ok {
		return nil, errors.Wrapf(errors.ErrModel, "invalid type: %T", obj.Value())
	}
	return rev, nil
}

// NewElectionRulesBucket is the persistent bucket for ElectionRules .
type ElectionRulesBucket struct {
	orm.VersioningBucket
}

// NewElectionRulesBucket returns a bucket for managing election rules.
func NewElectionRulesBucket() *ElectionRulesBucket {
	b := migration.NewBucket(packageName, "electnrule", orm.NewSimpleObj(nil, &ElectionRule{}))
	return &ElectionRulesBucket{
		VersioningBucket: orm.WithVersioning(orm.WithSeqIDGenerator(b, "id")),
	}
}

func asElectionRule(obj orm.Object) (*ElectionRule, error) {
	rev, ok := obj.Value().(*ElectionRule)
	if !ok {
		return nil, errors.Wrapf(errors.ErrModel, "invalid type: %T", obj.Value())
	}
	return rev, nil
}

// ProposalBucket is the persistent bucket for governance proposal objects.
type ProposalBucket struct {
	orm.IDGenBucket
}

const indexNameAuthor = "author"

// NewProposalBucket returns a bucket for managing electorate.
func NewProposalBucket() *ProposalBucket {
	b := migration.NewBucket(packageName, "proposal", orm.NewSimpleObj(nil, &Proposal{})).
		WithIndex(indexNameAuthor, indexAuthor, false)
	return &ProposalBucket{
		IDGenBucket: orm.WithSeqIDGenerator(b, "id"),
	}
}

func indexAuthor(obj orm.Object) ([]byte, error) {
	p, err := asProposal(obj)
	if err != nil {
		return nil, err
	}
	return p.Common.Author, nil
}

// GetProposal loads the proposal for the given id. If it does not exist then ErrNotFound is returned.
func (b *ProposalBucket) GetProposal(db weave.KVStore, id []byte) (*Proposal, error) {
	obj, err := b.Get(db, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load proposal")
	}
	return asProposal(obj)
}

func asProposal(obj orm.Object) (*Proposal, error) {
	if obj == nil || obj.Value() == nil {
		return nil, errors.Wrap(errors.ErrNotFound, "unknown id")
	}
	rev, ok := obj.Value().(*Proposal)
	if !ok {
		return nil, errors.Wrapf(errors.ErrModel, "invalid type: %T", obj.Value())
	}
	return rev, nil
}

// Update stores the given proposal and id in the persistence store.
func (b *ProposalBucket) Update(db weave.KVStore, id []byte, obj *Proposal) error {
	if err := b.Save(db, orm.NewSimpleObj(id, obj)); err != nil {
		return errors.Wrap(err, "failed to save")
	}
	return nil
}

const (
	indexNameProposal = "proposal"
	indexNameElector  = "elector"
)

// VoteBucket is the persistence bucket for votes.
type VoteBucket struct {
	orm.Bucket
}

// NewVoteBucket returns a bucket for managing electorate.
func NewVoteBucket() *VoteBucket {
	b := migration.NewBucket(packageName, "vote", orm.NewSimpleObj(nil, &Vote{})).
		WithIndex(indexNameProposal, indexProposal, false).
		WithIndex(indexNameElector, indexElector, false)
	return &VoteBucket{
		Bucket: b,
	}
}

func indexElector(obj orm.Object) (bytes []byte, e error) {
	if obj == nil {
		return nil, errors.Wrap(errors.ErrHuman, "cannot take index of nil")
	}
	v, ok := obj.Value().(*Vote)
	if !ok {
		return nil, errors.Wrap(errors.ErrHuman, "Can only take index of Vote")
	}
	return v.Elector.Address, nil
}

func indexProposal(obj orm.Object) (bytes []byte, e error) {
	if obj == nil {
		return nil, errors.Wrap(errors.ErrHuman, "cannot take index of nil")
	}
	compositeKey := obj.Key()
	if len(compositeKey) <= weave.AddressLength {
		return nil, errors.Wrap(errors.ErrInput, "unsupported key type")
	}
	proposalID := compositeKey[weave.AddressLength:]
	return proposalID, nil
}

// Build creates the orm object without storing it.
func (b *VoteBucket) Build(db weave.KVStore, proposalID []byte, vote Vote) orm.Object {
	compositeKey := compositeKey(proposalID, vote.Elector.Address)
	return orm.NewSimpleObj(compositeKey, &vote)
}

func compositeKey(proposalID []byte, address weave.Address) []byte {
	return append(address, proposalID...)
}

// HasVoted checks the bucket if any vote matching elector address and proposal id was stored.
func (b *VoteBucket) HasVoted(db weave.KVStore, proposalID []byte, addr weave.Address) (bool, error) {
	obj, err := b.Get(db, compositeKey(proposalID, addr))
	if err != nil {
		return false, errors.Wrap(err, "failed to load vote")
	}
	return obj != nil && obj.Value() != nil, nil
}

// GetVote loads the vote from the archive for the given proposal id and elector address.
// returns `errors.ErrNotFound` when not exists.
func (b *VoteBucket) GetVote(db weave.KVStore, proposalID []byte, addr weave.Address) (*Vote, error) {
	obj, err := b.Get(db, compositeKey(proposalID, addr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to load vote")
	}
	if obj == nil || obj.Value() == nil {
		return nil, errors.Wrap(errors.ErrNotFound, "unknown id")
	}
	v, ok := obj.Value().(*Vote)
	if !ok {
		return nil, errors.Wrapf(errors.ErrModel, "invalid type: %T", obj.Value())
	}
	return v, nil
}
