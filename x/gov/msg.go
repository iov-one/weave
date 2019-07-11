package gov

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &CreateProposalMsg{}, migration.NoModification)
	migration.MustRegister(1, &VoteMsg{}, migration.NoModification)
	migration.MustRegister(1, &TallyMsg{}, migration.NoModification)
	migration.MustRegister(1, &DeleteProposalMsg{}, migration.NoModification)
	migration.MustRegister(1, &UpdateElectionRuleMsg{}, migration.NoModification)
	migration.MustRegister(1, &UpdateElectorateMsg{}, migration.NoModification)
}

var _ weave.Msg = (*CreateProposalMsg)(nil)

func (CreateProposalMsg) Path() string {
	return "gov/create_proposal"
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

var _ weave.Msg = (*DeleteProposalMsg)(nil)

func (DeleteProposalMsg) Path() string {
	return "gov/delete_proposal"
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

var _ weave.Msg = (*VoteMsg)(nil)

func (VoteMsg) Path() string {
	return "gov/vote"
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

var _ weave.Msg = (*TallyMsg)(nil)

func (TallyMsg) Path() string {
	return "gov/tally"
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

var _ weave.Msg = (*UpdateElectionRuleMsg)(nil)

func (UpdateElectionRuleMsg) Path() string {
	return "gov/update_election_rule"
}

func (m UpdateElectionRuleMsg) Validate() error {
	var errs error

	if err := m.Metadata.Validate(); err != nil {
		errs = errors.Append(errs, errors.Wrap(err, "invalid metadata"))
	}
	if len(m.ElectionRuleID) == 0 {
		errs = errors.Append(errs, errors.Wrap(errors.ErrEmpty, "id"))
	}
	if m.VotingPeriod.Duration() < minVotingPeriod {
		errs = errors.Append(errs, errors.Wrapf(errors.ErrInput, "min %s", minVotingPeriod))
	}
	if m.VotingPeriod.Duration() > maxVotingPeriod {
		errs = errors.Append(errs, errors.Wrapf(errors.ErrInput, "max %s", maxVotingPeriod))
	}
	if m.Quorum != nil {
		if err := m.Quorum.Validate(); err != nil {
			errs = errors.Append(errs, errors.Wrap(err, "quorum"))
		}
	}
	if err := m.Threshold.Validate(); err != nil {
		errs = errors.Append(errs, errors.Wrap(err, "threshold"))
	}
	return errs
}

var _ weave.Msg = (*CreateTextResolutionMsg)(nil)

func (CreateTextResolutionMsg) Path() string {
	return "gov/create_text_resolution"
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

var _ weave.Msg = (*UpdateElectorateMsg)(nil)

func (UpdateElectorateMsg) Path() string {
	return "gov/update_electorate"
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
