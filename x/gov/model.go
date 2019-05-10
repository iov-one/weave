package gov

import (
	"math/big"
	"regexp"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

const maxElectors = 2000

var validTitle = regexp.MustCompile(`^[a-zA-Z0-9 _.-]{4,128}$`).MatchString

func init() {
	migration.MustRegister(1, &Electorate{}, migration.NoModification)
	migration.MustRegister(1, &ElectionRule{}, migration.NoModification)
	migration.MustRegister(1, &Proposal{}, migration.NoModification)
	migration.MustRegister(1, &Vote{}, migration.NoModification)
}

func (m *Electorate) SetVersion(v uint32) {
	m.Version = v
}

func (m Electorate) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "invalid metadata")
	}

	switch n := len(m.Electors); {
	case n == 0:
		return errors.Wrap(errors.ErrInput, "electors must not be empty")
	case n > maxElectors:
		return errors.Wrapf(errors.ErrInput, "electors must not exceed: %d", maxElectors)
	case !validTitle(m.Title):
		return errors.Wrapf(errors.ErrInput, "title: %q", m.Title)
	case len(m.UpdateElectionRuleID) == 0:
		return errors.Wrapf(errors.ErrEmpty, "update election rule id")
	}

	var totalWeight uint64
	for _, v := range m.Electors {
		if err := v.Validate(); err != nil {
			return err
		}
		totalWeight += uint64(v.Weight)
	}
	if diff := len(m.Electors) - newMerger(m.Electors).size(); diff != 0 {
		return errors.Wrapf(errors.ErrInput, "duplicate electors: %d", diff)
	}
	if m.TotalElectorateWeight != totalWeight {
		return errors.Wrap(errors.ErrInput, "total weight does not match sum")
	}
	if err := m.Admin.Validate(); err != nil {
		return errors.Wrap(err, "admin")
	}
	return nil
}

func (m Electorate) Copy() orm.CloneableData {
	p := make([]Elector, 0, len(m.Electors))
	copy(p, m.Electors)
	return &Electorate{
		Title:    m.Title,
		Electors: p,
		Version:  m.Version,
	}
}

// Weight return the weight for the given address is in the electors list and an ok flag which
// is true when the address exists in the electors list only.
func (m Electorate) Elector(a weave.Address) (*Elector, bool) {
	for _, v := range m.Electors {
		if v.Address.Equals(a) {
			return &v, true
		}
	}
	return nil, false
}

const maxWeight = 1<<16 - 1

func (m Elector) Validate() error {
	switch {
	case m.Weight > maxWeight:
		return errors.Wrap(errors.ErrInput, "must not be greater max weight")
	case m.Weight == 0:
		return errors.Wrap(errors.ErrInput, "weight must not be empty")
	}
	return m.Address.Validate()
}

const (
	minVotingPeriodHours = 1
	maxVotingPeriodHours = 4 * 7 * 24 // 4 weeks
)

func (m ElectionRule) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "invalid metadata")
	}

	switch {
	case !validTitle(m.Title):
		return errors.Wrapf(errors.ErrInput, "title: %q", m.Title)
	case m.VotingPeriodHours < minVotingPeriodHours:
		return errors.Wrapf(errors.ErrInput, "min hours: %d", minVotingPeriodHours)
	case m.VotingPeriodHours > maxVotingPeriodHours:
		return errors.Wrapf(errors.ErrInput, "max hours: %d", maxVotingPeriodHours)
	}
	if err := m.Admin.Validate(); err != nil {
		return errors.Wrap(err, "admin")
	}
	if err := m.Threshold.Validate(); err != nil {
		return errors.Wrap(err, "threshold")
	}
	if m.Quorum != nil {
		if err := m.Quorum.Validate(); err != nil {
			return errors.Wrap(err, "quorum")
		}
	}
	return nil
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
		return errors.Wrap(errors.ErrInput, "numerator must not be 0")
	case m.Denominator == 0:
		return errors.Wrap(errors.ErrInput, "denominator must not be 0")
	case uint64(m.Numerator)*2 < uint64(m.Denominator):
		return errors.Wrap(errors.ErrInput, "must not be lower 0.5")
	case m.Numerator > m.Denominator:
		return errors.Wrap(errors.ErrInput, "must not be greater 1")
	}
	return nil
}

const (
	minDescriptionLength    = 3
	maxDescriptionLength    = 5000
	maxFutureStartTimeHours = 7 * 24 * time.Hour // 1 week
)

func (m *Proposal) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "invalid metadata")
	}

	switch {
	case m.Result == Proposal_Empty:
		return errors.Wrap(errors.ErrInput, "invalid result value")
	case m.Status == Proposal_Invalid:
		return errors.Wrap(errors.ErrInput, "invalid status")
	case m.VotingStartTime >= m.VotingEndTime:
		return errors.Wrap(errors.ErrInput, "start time must be before end time")
	case m.VotingStartTime <= m.SubmissionTime:
		return errors.Wrap(errors.ErrInput, "start time must be after submission time")
	case len(m.Author) == 0:
		return errors.Wrap(errors.ErrInput, "author required")
	case len(m.Description) < minDescriptionLength:
		return errors.Wrapf(errors.ErrInput, "description length lower than minimum of: %d", minDescriptionLength)
	case len(m.Description) > maxDescriptionLength:
		return errors.Wrapf(errors.ErrInput, "description length exceeds: %d", maxDescriptionLength)
	case len(m.ElectionRuleID) == 0:
		return errors.Wrap(errors.ErrInput, "empty election rules id")
	case !validTitle(m.Title):
		return errors.Wrapf(errors.ErrInput, "title: %q", m.Title)
	}
	if err := m.ElectorateRef.Validate(); err != nil {
		return errors.Wrap(err, "electorate reference")
	}
	// validate details
	switch m.Type {
	case Proposal_Text:
	case Proposal_UpdateElectorate:
		if err := m.GetElectorateUpdateDetails().Validate(); err != nil {
			return err
		}
	default:
		return errors.Wrapf(errors.ErrState, "unsupported type: %v", m.Type)
	}
	return m.VoteState.Validate()
}

func (m Proposal) Copy() orm.CloneableData {
	electionRuleID := make([]byte, 0, len(m.ElectionRuleID))
	copy(electionRuleID, m.ElectionRuleID)
	return &Proposal{
		Title:           m.Title,
		Description:     m.Description,
		ElectionRuleID:  electionRuleID,
		ElectorateRef:   orm.VersionedIDRef{ID: m.ElectorateRef.ID, Version: m.ElectorateRef.Version},
		VotingStartTime: m.VotingStartTime,
		VotingEndTime:   m.VotingEndTime,
		SubmissionTime:  m.SubmissionTime,
		Author:          m.Author,
		VoteState:       m.VoteState,
		Status:          m.Status,
		Result:          m.Result,
	}
}

// CountVote updates the intermediate tally result by adding the new vote weight.
func (m *Proposal) CountVote(vote Vote) error {
	oldTotal := m.VoteState.TotalVotes()
	switch vote.Voted {
	case VoteOption_Yes:
		m.VoteState.TotalYes += uint64(vote.Elector.Weight)
	case VoteOption_No:
		m.VoteState.TotalNo += uint64(vote.Elector.Weight)
	case VoteOption_Abstain:
		m.VoteState.TotalAbstain += uint64(vote.Elector.Weight)
	default:
		return errors.Wrapf(errors.ErrInput, "%q", m.String())
	}
	if m.VoteState.TotalVotes() <= oldTotal {
		return errors.Wrap(errors.ErrHuman, "sanity overflow check failed")
	}
	return nil
}

// UndoCountVote updates the intermediate tally result by subtracting the given vote weight.
func (m *Proposal) UndoCountVote(vote Vote) error {
	oldTotal := m.VoteState.TotalVotes()
	switch vote.Voted {
	case VoteOption_Yes:
		m.VoteState.TotalYes -= uint64(vote.Elector.Weight)
	case VoteOption_No:
		m.VoteState.TotalNo -= uint64(vote.Elector.Weight)
	case VoteOption_Abstain:
		m.VoteState.TotalAbstain -= uint64(vote.Elector.Weight)
	default:
		return errors.Wrapf(errors.ErrInput, "%q", m.String())
	}
	if m.VoteState.TotalVotes() >= oldTotal {
		return errors.Wrap(errors.ErrHuman, "sanity overflow check failed")
	}
	return nil
}

// Tally calls the final calculation on the votes and sets the status of the proposal according to the
// election rules threshold.
func (m *Proposal) Tally() error {
	if m.Result != Proposal_Undefined {
		return errors.Wrapf(errors.ErrState, "result exists: %q", m.Result.String())
	}
	if m.Status != Proposal_Submitted {
		return errors.Wrapf(errors.ErrState, "unexpected status: %q", m.Status.String())
	}
	if m.VoteState.Accepted() {
		m.Result = Proposal_Accepted
	} else {
		m.Result = Proposal_Rejected
	}
	m.Status = Proposal_Closed
	return nil
}

func NewTallyResult(quorum *Fraction, threshold Fraction, totalElectorateWeight uint64) TallyResult {
	return TallyResult{
		Quorum:                quorum,
		Threshold:             threshold,
		TotalElectorateWeight: totalElectorateWeight,
	}
}

//Accepted returns the result of the calculation if a proposal got enough votes or not.
func (m TallyResult) Accepted() bool {
	if m.TotalYes == m.TotalElectorateWeight { // handles 1/1 threshold
		return true
	}

	total := m.TotalVotes()
	bTotalElectorateWeight := new(big.Int).SetUint64(m.TotalElectorateWeight)
	bBaseWeight := bTotalElectorateWeight
	if m.Quorum != nil {
		// new base = total Yes + total No
		bBaseWeight = new(big.Int).Add(new(big.Int).SetUint64(m.TotalYes), new(big.Int).SetUint64(m.TotalNo))

		if total != m.TotalElectorateWeight { // handles non 1/1 quorums only
			// quorum reached when
			// totalVotes * quorumDenominator > electorate * quorumNumerator
			bTotalVotes := new(big.Int).SetUint64(total)
			p1 := new(big.Int).Mul(bTotalVotes, big.NewInt(int64(m.Quorum.Denominator)))
			p2 := new(big.Int).Mul(bTotalElectorateWeight, big.NewInt(int64(m.Quorum.Numerator)))
			if p1.Cmp(p2) < 1 {
				return false
			}
		}
	}

	// (yes * denominator) > (base * numerator) with base total electorate weight or YesNo votes in case of quorum set
	bTotalYes := new(big.Int).SetUint64(m.TotalYes)
	p1 := new(big.Int).Mul(bTotalYes, big.NewInt(int64(m.Threshold.Denominator)))
	p2 := new(big.Int).Mul(bBaseWeight, big.NewInt(int64(m.Threshold.Numerator)))
	return p1.Cmp(p2) > 0
}

// TotalVotes returns the sum of yes, no, abstain votes weights.
func (m TallyResult) TotalVotes() uint64 {
	return m.TotalYes + m.TotalNo + m.TotalAbstain
}

func (m TallyResult) Validate() error {
	switch {
	case m.Quorum != nil:
		if err := m.Quorum.Validate(); err != nil {
			return errors.Wrap(errors.ErrState, "quorum")
		}
	case m.TotalElectorateWeight == 0:
		return errors.Wrap(errors.ErrState, "totalElectorateWeight")
	case m.TotalVotes() > m.TotalElectorateWeight:
		return errors.Wrap(errors.ErrState, "votes must not exceed totalElectorateWeight")
	}
	if err := m.Threshold.Validate(); err != nil {
		return errors.Wrap(errors.ErrState, "threshold")
	}
	return nil
}

// validate vote object contains valid elector and voted option
func (m Vote) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "invalid metadata")
	}
	if err := m.Elector.Validate(); err != nil {
		return errors.Wrap(err, "invalid elector")
	}
	if m.Voted == VoteOption_Invalid {
		return errors.Wrap(errors.ErrInput, "invalid vote option")
	}
	return nil
}

func (m Vote) Copy() orm.CloneableData {
	return &Vote{
		Elector: m.Elector,
		Voted:   m.Voted,
	}
}

func (m *ElectorateUpdatePayload) Validate() error {
	if m == nil {
		return errors.ErrEmpty
	}
	return ElectorsDiff(m.DiffElectors).Validate()
}

// DiffElectors contains the changes that should be applied. Adding an address should have a positive weight, removing
// with weight=0.
type ElectorsDiff []Elector

func (e ElectorsDiff) Validate() error {
	if len(e) == 0 {
		return errors.Wrap(errors.ErrEmpty, "electors")
	}
	for i, v := range e {
		if v.Weight > maxWeight {
			return errors.Wrap(errors.ErrInput, "must not be greater max weight")
		}
		if err := v.Address.Validate(); err != nil {
			return errors.Wrapf(err, "address at position: %d", i)
		}
	}
	return nil
}
