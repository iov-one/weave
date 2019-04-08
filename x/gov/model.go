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
	switch n := len(m.Electors); {
	case n == 0:
		return errors.Wrap(errors.ErrInvalidInput, "electors must not be empty")
	case n > maxElectors:
		return errors.Wrap(errors.ErrInvalidInput, fmt.Sprintf("electors must not exceed: %d", maxElectors))
	case !validTitle(m.Title):
		return errors.Wrap(errors.ErrInvalidInput, fmt.Sprintf("title: %q", m.Title))
	}

	var totalWeight uint64
	index := make(map[string]struct{}) // check for duplicate votes
	for _, v := range m.Electors {
		if err := v.Validate(); err != nil {
			return err
		}
		totalWeight += uint64(v.Weight)
		if _, exists := index[v.Signature.String()]; exists {
			return errors.Wrap(errors.ErrInvalidInput, "duplicate elector entry")
		}
		index[v.Signature.String()] = struct{}{}
	}

	if m.TotalWeightElectorate != totalWeight {
		return errors.Wrap(errors.ErrInvalidInput, "total weight does not match sum")
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
	switch {
	case !validTitle(m.Title):
		return errors.Wrap(errors.ErrInvalidInput, fmt.Sprintf("title: %q", m.Title))
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
	case uint64(m.Numerator)*2 < uint64(m.Denominator):
		return errors.Wrap(errors.ErrInvalidInput, "must not be lower 0.5")
	case m.Numerator > m.Denominator:
		return errors.Wrap(errors.ErrInvalidInput, "must not be greater 1")
	}
	return nil
}

const (
	minDescriptionLength = 3
	maxDescriptionLength = 5000
)

func (m *TextProposal) Validate() error {
	switch {
	case !validTitle(m.Title):
		return errors.Wrap(errors.ErrInvalidInput, fmt.Sprintf("title: %q", m.Title))
	case m.Status == TextProposal_Invalid:
		return errors.Wrap(errors.ErrInvalidInput, "invalid status")
	case !m.VotingStartTime.Time().Before(m.VotingEndTime.Time()):
		return errors.Wrap(errors.ErrInvalidInput, "start time must be before end time")
	case !m.VotingStartTime.Time().After(m.SubmissionTime.Time()):
		return errors.Wrap(errors.ErrInvalidInput, "start time must be after submission time")
	case len(m.Author) == 0:
		return errors.Wrap(errors.ErrInvalidInput, "author required")
	case len(m.Description) < minDescriptionLength:
		return errors.Wrap(errors.ErrInvalidInput, fmt.Sprintf("description length lower than minimum of: %d", minDescriptionLength))
	case len(m.Description) > maxDescriptionLength:
		return errors.Wrap(errors.ErrInvalidInput, fmt.Sprintf("description length exceeds: %d", maxDescriptionLength))
	case len(m.ElectorateID) == 0:
		return errors.Wrap(errors.ErrInvalidInput, "empty electorate id")
	case len(m.ElectionRuleID) == 0:
		return errors.Wrap(errors.ErrInvalidInput, "empty election rules id")
	}

	// check for duplicate votes
	index := make(map[string]struct{})
	for i, v := range m.Votes {
		err := v.Elector.Validate()
		switch {
		case err != nil:
			return errors.Wrap(err, fmt.Sprintf("invalid elector in vote: %d", i))
		case v.Voted == VoteOption_Invalid:
			return errors.Wrap(errors.ErrInvalidInput, fmt.Sprintf("invalid option in vote: %d", i))
		}
		if _, exists := index[v.Elector.Signature.String()]; exists {
			return errors.Wrap(errors.ErrInvalidInput, fmt.Sprintf("duplicate vote for address: %s", v.Elector.Signature.String()))
		}
		index[v.Elector.Signature.String()] = struct{}{}
	}

	return nil
}

func (m TextProposal) Copy() orm.CloneableData {
	votes := make([]*Vote, 0, len(m.Votes))
	copy(votes, m.Votes)
	electionRuleID := make([]byte, 0, len(m.ElectionRuleID))
	copy(electionRuleID, m.ElectionRuleID)
	electorateID := make([]byte, 0, len(m.ElectorateID))
	copy(electorateID, m.ElectorateID)
	return &TextProposal{
		Title:           m.Title,
		Description:     m.Description,
		ElectionRuleID:  electionRuleID,
		ElectorateID:    electorateID,
		VotingStartTime: m.VotingStartTime,
		VotingEndTime:   m.VotingEndTime,
		SubmissionTime:  m.SubmissionTime,
		Author:          m.Author,
		Votes:           votes,
		VoteResult:      m.VoteResult,
		Status:          m.Status,
	}
}

// Vote updates the intermediate tally result with the new vote and stores the elector in the
// voter archive.
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

// Tally calls the final calculation on the votes and sets the status of the proposal according to the
// election rules threshold.
func (m *TextProposal) Tally() error {
	if m.VoteResult.Accepted() {
		m.Status = TextProposal_Accepted
	} else {
		m.Status = TextProposal_Rejected
	}
	return nil
}

// HasVoted returns if the given address has been in the voter archive for this proposal.
func (m TextProposal) HasVoted(a weave.Address) bool {
	for _, v := range m.Votes {
		if v.Elector.Signature.Equals(a) {
			return true
		}
	}
	return false
}

// Accepted returns the result of the `(yes*denominator) > (numerator*total_electors_weight)` calculation.
func (m TallyResult) Accepted() bool {
	return uint64(m.TotalYes)*uint64(m.Threshold.Denominator) > m.TotalWeightElectorate*uint64(m.Threshold.Numerator)
}
