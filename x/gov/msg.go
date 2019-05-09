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
		return errors.Wrap(errors.ErrInput, "empty election rules id")
	}
	return validateCreateProposal(&m)
}

func (DeleteProposalMsg) Path() string {
	return pathDeleteTextProposalMsg
}

func (m DeleteProposalMsg) Validate() error {
	if len(m.ID) == 0 {
		return errors.Wrap(errors.ErrInput, "empty proposal id")
	}
	return nil
}

func (VoteMsg) Path() string {
	return pathVoteMsg
}

func (m VoteMsg) Validate() error {
	if m.Selected != VoteOption_Yes && m.Selected != VoteOption_No && m.Selected != VoteOption_Abstain {
		return errors.Wrap(errors.ErrInput, "invalid option")
	}
	if len(m.ProposalID) == 0 {
		return errors.Wrap(errors.ErrInput, "empty proposal id")
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
		return errors.Wrap(errors.ErrInput, "empty proposal id")
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
		return errors.Wrapf(errors.ErrInput, "min hours: %d", minVotingPeriodHours)
	case m.VotingPeriodHours > maxVotingPeriodHours:
		return errors.Wrapf(errors.ErrInput, "max hours: %d", maxVotingPeriodHours)
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
	case len(m.DiffElectors) == 0:
		return errors.Wrap(errors.ErrEmpty, "electors")
	}
	for i, v := range m.DiffElectors {
		if v.Weight > maxWeight {
			return errors.Wrap(errors.ErrInput, "must not be greater max weight")
		}
		if err := v.Address.Validate(); err != nil {
			return errors.Wrapf(err, "address at position: %d", i)
		}
	}
	return nil
}

func (CreateElectorateUpdateProposalMsg) Path() string {
	return pathCreateElectorateUpdateProposalMsg
}

func (m CreateElectorateUpdateProposalMsg) Validate() error {
	for i, v := range m.DiffElectors {
		if v.Weight > maxWeight {
			return errors.Wrap(errors.ErrInput, "must not be greater max weight")
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
	switch {
	case len(m.GetElectorateID()) == 0:
		return errors.Wrap(errors.ErrInput, "empty electorate id")
	case m.GetStartTime() == 0:
		return errors.Wrap(errors.ErrInput, "empty start time")
	case !validTitle(m.GetTitle()):
		return errors.Wrapf(errors.ErrInput, "title: %q", m.GetTitle())
	case len(m.GetDescription()) < minDescriptionLength:
		return errors.Wrapf(errors.ErrInput, "description length lower than minimum of: %d", minDescriptionLength)
	case len(m.GetDescription()) > maxDescriptionLength:
		return errors.Wrapf(errors.ErrInput, "description length exceeds: %d", maxDescriptionLength)
	}
	if err := m.GetStartTime().Validate(); err != nil {
		return errors.Wrap(err, "start time")
	}
	if m.GetAuthor() != nil {
		if err := m.GetAuthor().Validate(); err != nil {
			return errors.Wrap(err, "author")
		}
	}
	return nil
}
