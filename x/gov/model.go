package gov

import (
	"fmt"
	"regexp"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
)

var validTitle = regexp.MustCompile(`^[a-zA-Z0-9 _.-]{4,128}$`).MatchString

const maxParticipants = 2000

func (m Electorate) Validate() error {
	if !validTitle(m.Title) {
		return errors.Wrap(errors.ErrInvalidInput, fmt.Sprintf("title: %q", m.Title))
	}
	switch n := len(m.Participants); {
	case n == 0:
		return errors.Wrap(errors.ErrInvalidInput, "participants must not be empty")
	case n > maxParticipants:
		return errors.Wrap(errors.ErrInvalidInput, fmt.Sprintf("participants must not exceed: %d", maxParticipants))
	}
	for _, v := range m.Participants {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (m Electorate) Copy() orm.CloneableData {
	p := make([]Participant, 0, len(m.Participants))
	copy(p, m.Participants)
	return &Electorate{
		Title:        m.Title,
		Participants: p,
	}
}

const maxWeight = 2 ^ 16 - 1

func (m Participant) Validate() error {
	switch {
	case m.Weight > maxWeight:
		return errors.Wrap(errors.ErrInvalidInput, "must not be greater max weight")
	case m.Weight == 0:
		return errors.Wrap(errors.ErrInvalidInput, "weight must not be empty")
	}
	return m.Signature.Validate()
}

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

const (
	minVotingPeriodHours = 1
	maxVotingPeriodHours = 4 * 7 * 24 // 4 weeks
)

func (m ElectionRule) Validate() error {
	if !validTitle(m.Title) {
		return errors.Wrap(errors.ErrInvalidInput, fmt.Sprintf("title: %q", m.Title))
	}
	switch {
	case m.VotingPeriodHours < minVotingPeriodHours:
		return errors.Wrap(errors.ErrInvalidInput, fmt.Sprintf("min hours: %d", minVotingPeriodHours))
	case m.VotingPeriodHours > maxVotingPeriodHours:
		return errors.Wrap(errors.ErrInvalidInput, fmt.Sprintf("max hours: %d", maxVotingPeriodHours))
	}
	return m.Threshold.Validate()
}

func (m ElectionRule) Copy() orm.CloneableData {
	return &ElectionRule{
		Title:             m.Title,
		VotingPeriodHours: m.VotingPeriodHours,
		Threshold:         m.Threshold,
	}
}

func (m Fraction) Validate() error {
	switch {
	case m.Numerator == 0:
		return errors.Wrap(errors.ErrInvalidInput, "numerator must not be 0")
	case m.Denominator == 0:
		return errors.Wrap(errors.ErrInvalidInput, "denominator must not be 0")
	case m.Numerator*2 < m.Denominator:
		return errors.Wrap(errors.ErrInvalidInput, "must not be lower 0.5")
	case m.Numerator/m.Denominator > 1:
		return errors.Wrap(errors.ErrInvalidInput, "must not be greater 1")
	}
	return nil
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
