package gov

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

const (
	pathCreateTextProposalMsg             = "gov/create/text"
	pathCreateElectorateUpdateProposalMsg = "gov/create/electorateUpdate"
	pathDeleteTextProposalMsg             = "gov/delete"
	pathVoteMsg                           = "gov/vote"
	pathTallyMsg                          = "gov/tally"
	pathUpdateElectorateMsg               = "gov/electorate/update"
	pathUpdateElectionRulesMsg            = "gov/electionRules/update"
)

var _ weave.Msg = (*CreateTextProposalMsg)(nil)
var _ weave.Msg = (*VoteMsg)(nil)
var _ weave.Msg = (*TallyMsg)(nil)
var _ weave.Msg = (*DeleteProposalMsg)(nil)
var _ weave.Msg = (*CreateElectorateUpdateProposalMsg)(nil)

func (CreateTextProposalMsg) Path() string {
	return pathCreateTextProposalMsg
}

func (m CreateTextProposalMsg) Validate() error {
	if len(m.ElectionRuleID) == 0 {
		return errors.Wrap(errors.ErrInvalidInput, "empty election rules id")
	}
	return validateCreateProposal(&m)
}

func (DeleteProposalMsg) Path() string {
	return pathDeleteTextProposalMsg
}

func (m DeleteProposalMsg) Validate() error {
	if len(m.ID) == 0 {
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

func (CreateElectorateUpdateProposalMsg) Path() string {
	return pathCreateElectorateUpdateProposalMsg
}

func (m CreateElectorateUpdateProposalMsg) Validate() error {
	for i, v := range m.DiffElectors {
		if v.Weight > maxWeight {
			return errors.Wrap(errors.ErrInvalidInput, "must not be greater max weight")
		}
		if err := v.Address.Validate(); err != nil {
			return errors.Wrapf(err, "address at position: %d", i)
		}
	}
	return validateCreateProposal(&m)
}

type commonCreateProposalData interface {
	GetTitle() string
	GetDescription() string
	GetElectorateID() []byte
	GetStartTime() weave.UnixTime
	GetAuthor() weave.Address
}

func validateCreateProposal(m commonCreateProposalData) error {
	err := m.GetAuthor().Validate()
	switch {
	case len(m.GetElectorateID()) == 0:
		return errors.Wrap(errors.ErrInvalidInput, "empty electorate id")
	case m.GetStartTime() == 0:
		return errors.Wrap(errors.ErrInvalidInput, "empty start time")
	case m.GetAuthor() != nil && err != nil:
		return errors.Wrap(err, "invalid author")
	case !validTitle(m.GetTitle()):
		return errors.Wrapf(errors.ErrInvalidInput, "title: %q", m.GetTitle())
	case len(m.GetDescription()) < minDescriptionLength:
		return errors.Wrapf(errors.ErrInvalidInput, "description length lower than minimum of: %d", minDescriptionLength)
	case len(m.GetDescription()) > maxDescriptionLength:
		return errors.Wrapf(errors.ErrInvalidInput, "description length exceeds: %d", maxDescriptionLength)
	}
	return m.GetStartTime().Validate()

}
