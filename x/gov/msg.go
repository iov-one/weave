package gov

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

const (
	pathCreateTextProposalMsg  = "gov/create"
	pathDeleteTextProposalMsg  = "gov/delete"
	pathVoteMsg                = "gov/vote"
	pathTallyMsg               = "gov/tally"
	pathUpdateElectorateMsg    = "gov/electorate/update"
	pathUpdateElectionRulesMsg = "gov/electionRules/update"
)

var _ weave.Msg = (*CreateTextProposalMsg)(nil)
var _ weave.Msg = (*VoteMsg)(nil)
var _ weave.Msg = (*TallyMsg)(nil)
var _ weave.Msg = (*DeleteTextProposalMsg)(nil)

func (CreateTextProposalMsg) Path() string {
	return pathCreateTextProposalMsg
}

func (m CreateTextProposalMsg) Validate() error {
	err := m.Author.Validate()
	switch {
	case len(m.ElectorateID) == 0:
		return errors.Wrap(errors.ErrInvalidInput, "empty electorate id")
	case len(m.ElectionRuleID) == 0:
		return errors.Wrap(errors.ErrInvalidInput, "empty election rules id")
	case m.StartTime == 0:
		return errors.Wrap(errors.ErrInvalidInput, "empty start time")
	case m.Author != nil && err != nil:
		return errors.Wrap(err, "invalid author")
	case !validTitle(m.Title):
		return errors.Wrapf(errors.ErrInvalidInput, "title: %q", m.Title)
	case len(m.Description) < minDescriptionLength:
		return errors.Wrapf(errors.ErrInvalidInput, "description length lower than minimum of: %d", minDescriptionLength)
	case len(m.Description) > maxDescriptionLength:
		return errors.Wrapf(errors.ErrInvalidInput, "description length exceeds: %d", maxDescriptionLength)
	}
	return m.StartTime.Validate()
}

func (DeleteTextProposalMsg) Path() string {
	return pathDeleteTextProposalMsg
}

func (m DeleteTextProposalMsg) Validate() error {
	if len(m.Id) == 0 {
		return errors.Wrap(errors.ErrInvalidInput, "empty proposal id")
	}
	return nil
}

func (VoteMsg) Path() string {
	return pathVoteMsg
}

func (m VoteMsg) Validate() error {
	if m.Selected != VoteOption_Yes && m.Selected != VoteOption_No && m.Selected != VoteOption_Abstain {
		return errors.Wrap(errors.ErrInvalidInput, "invalid option")
	}
	if len(m.ProposalID) == 0 {
		return errors.Wrap(errors.ErrInvalidInput, "empty proposal id")
	}
	if err := m.Voter.Validate(); m.Voter != nil && err != nil {
		return errors.Wrap(err, "invalid voter")
	}
	return nil
}

func (TallyMsg) Path() string {
	return pathTallyMsg
}

func (m TallyMsg) Validate() error {
	if len(m.ProposalID) == 0 {
		return errors.Wrap(errors.ErrInvalidInput, "empty proposal id")
	}
	return nil
}

func (UpdateElectionRuleMsg) Path() string {
	return pathUpdateElectionRulesMsg
}

func (m UpdateElectionRuleMsg) Validate() error {
	switch {
	case len(m.ElectionRuleID) == 0:
		return errors.Wrap(errors.ErrEmpty, "id")
	case m.VotingPeriodHours < minVotingPeriodHours:
		return errors.Wrapf(errors.ErrInvalidInput, "min hours: %d", minVotingPeriodHours)
	case m.VotingPeriodHours > maxVotingPeriodHours:
		return errors.Wrapf(errors.ErrInvalidInput, "max hours: %d", maxVotingPeriodHours)
	}
	return m.Threshold.Validate()
}

func (UpdateElectorateMsg) Path() string {
	return pathUpdateElectorateMsg
}

func (m UpdateElectorateMsg) Validate() error {
	switch {
	case len(m.ElectorateID) == 0:
		return errors.Wrap(errors.ErrEmpty, "id")
	case len(m.Electors) == 0:
		return errors.Wrap(errors.ErrEmpty, "electors")
	case len(m.Electors) > maxElectors:
		return errors.Wrapf(errors.ErrInvalidInput, "electors must not exceed: %d", maxElectors)
	}
	index := map[string]struct{}{} // address index for duplicates
	for i, v := range m.Electors {
		if err := v.Validate(); err != nil {
			return errors.Wrapf(err, "elector %d", i)
		}
		index[v.Address.String()] = struct{}{}
	}
	if len(index) != len(m.Electors) {
		return errors.Wrap(errors.ErrInvalidInput, "duplicate addresses")
	}
	return nil
}
