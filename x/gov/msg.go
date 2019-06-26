package gov

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

const (
	pathCreateProposalMsg       = "gov/create_proposal"
	pathDeleteProposalMsg       = "gov/delete_proposal"
	pathVoteMsg                 = "gov/vote"
	pathTallyMsg                = "gov/tally"
	pathCreateTextResolutionMsg = "gov/create_text_resolution"
	pathUpdateElectorateMsg     = "gov/update_electorate"
	pathUpdateElectionRuleMsg   = "gov/update_election_rule"
)

func init() {
	migration.MustRegister(1, &CreateProposalMsg{}, migration.NoModification)
	migration.MustRegister(1, &VoteMsg{}, migration.NoModification)
	migration.MustRegister(1, &TallyMsg{}, migration.NoModification)
	migration.MustRegister(1, &DeleteProposalMsg{}, migration.NoModification)
	migration.MustRegister(1, &UpdateElectionRuleMsg{}, migration.NoModification)
	migration.MustRegister(1, &UpdateElectorateMsg{}, migration.NoModification)
}

func (CreateProposalMsg) Path() string {
	return pathCreateProposalMsg
}

func (m CreateProposalMsg) Validate() error {
	if err := m.GetMetadata().Validate(); err != nil {
		return errors.Wrap(err, "invalid metadata")
	}
	if len(m.RawOption) == 0 {
		return errors.Wrap(errors.ErrEmpty, "missing raw options")
	}
	if len(m.GetElectionRuleID()) == 0 {
		return errors.Wrap(errors.ErrInput, "empty election rules id")
	}
	if m.GetStartTime() == 0 {
		return errors.Wrap(errors.ErrInput, "empty start time")
	}
	if !validTitle(m.GetTitle()) {
		return errors.Wrapf(errors.ErrInput, "title: %q", m.GetTitle())
	}
	if len(m.GetDescription()) < minDescriptionLength {
		return errors.Wrapf(errors.ErrInput, "description length lower than minimum of: %d", minDescriptionLength)
	}
	if len(m.GetDescription()) > maxDescriptionLength {
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

func (DeleteProposalMsg) Path() string {
	return pathDeleteProposalMsg
}

func (m DeleteProposalMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "invalid metadata")
	}

	if len(m.ProposalID) != 8 {
		return errors.Wrap(errors.ErrInput, "proposal ids must be 8 bytes (sequence)")
	}
	return nil
}

func (VoteMsg) Path() string {
	return pathVoteMsg
}

func (m VoteMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "invalid metadata")
	}
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
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "invalid metadata")
	}

	if len(m.ProposalID) == 0 {
		return errors.Wrap(errors.ErrInput, "empty proposal id")
	}
	return nil
}

func (UpdateElectionRuleMsg) Path() string {
	return pathUpdateElectionRuleMsg
}

func (m UpdateElectionRuleMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "invalid metadata")
	}
	if len(m.ElectionRuleID) == 0 {
		return errors.Wrap(errors.ErrEmpty, "id")
	}
	if m.VotingPeriod.Duration() < minVotingPeriod {
		return errors.Wrapf(errors.ErrInput, "min %s", minVotingPeriod)
	}
	if m.VotingPeriod.Duration() > maxVotingPeriod {
		return errors.Wrapf(errors.ErrInput, "max %s", maxVotingPeriod)
	}
	return m.Threshold.Validate()
}

func (CreateTextResolutionMsg) Path() string {
	return pathCreateTextResolutionMsg
}

func (m CreateTextResolutionMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "invalid metadata")
	}
	if len(m.Resolution) == 0 {
		return errors.Wrap(errors.ErrEmpty, "resolution")
	}
	return nil
}

func (UpdateElectorateMsg) Path() string {
	return pathUpdateElectorateMsg
}

func (m UpdateElectorateMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "invalid metadata")
	}

	if len(m.ElectorateID) == 0 {
		return errors.Wrap(errors.ErrEmpty, "electorate id")
	}
	return ElectorsDiff(m.DiffElectors).Validate()
}
