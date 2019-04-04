package gov

import (
	"fmt"
	"regexp"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
)

var validTitle = regexp.MustCompile(`^[a-zA-Z0-9 _.-]{4,128}$`).MatchString

const maxElectors = 2000

func (m Electorate) Validate() error {
	if !validTitle(m.Title) {
		return errors.Wrap(errors.ErrInvalidInput, fmt.Sprintf("title: %q", m.Title))
	}
	switch n := len(m.Electors); {
	case n == 0:
		return errors.Wrap(errors.ErrInvalidInput, "electors must not be empty")
	case n > maxElectors:
		return errors.Wrap(errors.ErrInvalidInput, fmt.Sprintf("electors must not exceed: %d", maxElectors))
	}
	for _, v := range m.Electors {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (m Electorate) Copy() orm.CloneableData {
	p := make([]Elector, 0, len(m.Electors))
	copy(p, m.Electors)
	return &Electorate{
		Title:    m.Title,
		Electors: p,
	}
}

// Weight return the weight for the given address is in the electors list and an ok flag which
// is true when the address exists in the electors list only.
func (m Electorate) Elector(a weave.Address) (*Elector, bool) {
	for _, v := range m.Electors {
		if v.Signature.Equals(a) {
			return &v, true
		}
	}
	return nil, false
}

const maxWeight = 2 ^ 16 - 1

func (m Elector) Validate() error {
	switch {
	case m.Weight > maxWeight:
		return errors.Wrap(errors.ErrInvalidInput, "must not be greater max weight")
	case m.Weight == 0:
		return errors.Wrap(errors.ErrInvalidInput, "weight must not be empty")
	}
	return m.Signature.Validate()
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

func (m *TextProposal) Validate() error {
	// TODO impl
	return nil
}

func (m TextProposal) Copy() orm.CloneableData {
	// TODO impl
	return &m
}

func (m *TextProposal) Vote(voted VoteOption, elector Elector) error {
	switch voted {
	case VoteOption_Yes:
		m.VoteResult.TotalYes += elector.Weight
	case VoteOption_No:
		m.VoteResult.TotalNo += elector.Weight
	case VoteOption_Abstain:
		m.VoteResult.TotalAbstain += elector.Weight
	default:
		return errors.Wrap(errors.ErrInvalidInput, fmt.Sprintf("%q", m.String()))
	}
	m.Votes = append(m.Votes, &Vote{Elector: elector, Voted: voted})
	return nil
}

func (m TextProposal) HasVoted(a weave.Address) bool {
	for _, v := range m.Votes {
		if v.Elector.Signature.Equals(a) {
			return true
		}
	}
	return false
}
