package gov

import (
	"fmt"

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
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	if len(m.RawOption) == 0 {
		errs = errors.AppendField(errs, "RawOption", errors.ErrEmpty)
	}
	if len(m.ElectionRuleID) == 0 {
		errs = errors.AppendField(errs, "ElectionRuleID", errors.ErrInput)
	}
	if m.StartTime == 0 {
		errs = errors.Append(errs, errors.Field("StartTime", errors.ErrInput, "must not be zero"))
	} else {
		errs = errors.AppendField(errs, "StartTime", m.StartTime.Validate())
	}
	if !validTitle(m.Title) {
		errs = errors.AppendField(errs, "Title", errors.ErrInput)
	}
	if len(m.Description) < minDescriptionLength {
		errs = errors.Append(errs,
			errors.Field("Description", errors.ErrInput, "description length lower than minimum of %d", minDescriptionLength))
	} else if len(m.Description) > maxDescriptionLength {
		errs = errors.Append(errs,
			errors.Field("Description", errors.ErrInput, "description length exceeds the limit of %d", maxDescriptionLength))
	}
	if m.Author != nil {
		errs = errors.AppendField(errs, "Author", m.Author.Validate())
	}
	return errs
}

var _ weave.Msg = (*DeleteProposalMsg)(nil)

func (DeleteProposalMsg) Path() string {
	return "gov/delete_proposal"
}

func (m DeleteProposalMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	if len(m.ProposalID) != 8 {
		errs = errors.Append(errs, errors.Field("ProposalID", errors.ErrInput, "proposal ID must be 8 bytes (sequence)"))
	}
	return errs
}

var _ weave.Msg = (*VoteMsg)(nil)

func (VoteMsg) Path() string {
	return "gov/vote"
}

func (m VoteMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	if m.Selected != VoteOption_Yes && m.Selected != VoteOption_No && m.Selected != VoteOption_Abstain {
		errs = errors.AppendField(errs, "Selected", errors.ErrInput)
	}
	if len(m.ProposalID) == 0 {
		errs = errors.Append(errs, errors.Field("ProposalID", errors.ErrInput, "proposal ID is required"))
	}
	if m.Voter != nil {
		errs = errors.AppendField(errs, "Voter", m.Voter.Validate())
	}
	return errs
}

var _ weave.Msg = (*TallyMsg)(nil)

func (TallyMsg) Path() string {
	return "gov/tally"
}

func (m TallyMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	if len(m.ProposalID) == 0 {
		errs = errors.Append(errs, errors.Field("ProposalID", errors.ErrInput, "proposal ID is required"))
	}
	return errs
}

var _ weave.Msg = (*UpdateElectionRuleMsg)(nil)

func (UpdateElectionRuleMsg) Path() string {
	return "gov/update_election_rule"
}

func (m UpdateElectionRuleMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	if len(m.ElectionRuleID) == 0 {
		errs = errors.Append(errs, errors.Field("ElectionRuleID", errors.ErrEmpty, "election rule ID is required"))
	}
	if m.VotingPeriod.Duration() < minVotingPeriod {
		errs = errors.Append(errs, errors.Field("VotingPeriod", errors.ErrInput, "value must not be smaller than %s", minVotingPeriod))
	} else if m.VotingPeriod.Duration() > maxVotingPeriod {
		errs = errors.Append(errs, errors.Field("VotingPeriod", errors.ErrInput, "value must not be greater than %s", maxVotingPeriod))
	}
	if m.Quorum != nil {
		errs = errors.AppendField(errs, "Quorum", m.Quorum.Validate())
	}
	errs = errors.AppendField(errs, "Threshold", m.Threshold.Validate())
	return errs
}

var _ weave.Msg = (*CreateTextResolutionMsg)(nil)

func (CreateTextResolutionMsg) Path() string {
	return "gov/create_text_resolution"
}

func (m CreateTextResolutionMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	if len(m.Resolution) == 0 {
		errs = errors.AppendField(errs, "Resolution", errors.ErrEmpty)
	}
	return errs
}

var _ weave.Msg = (*UpdateElectorateMsg)(nil)

func (UpdateElectorateMsg) Path() string {
	return "gov/update_electorate"
}

func (m UpdateElectorateMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	if len(m.ElectorateID) == 0 {
		errs = errors.AppendField(errs, "ElectorateID", errors.ErrEmpty)
	}
	if len(m.DiffElectors) == 0 {
		errs = errors.AppendField(errs, "DiffElectors", errors.ErrEmpty)
	}
	for i, v := range m.DiffElectors {
		if v.Weight > maxWeight {
			errs = errors.Append(errs, errors.Field(fmt.Sprintf("DiffElectors.%d.Weight", i), errors.ErrInput, "must not be greater than max weight"))
		}
		errs = errors.AppendField(errs, fmt.Sprintf("DiffElectors.%d.Address", i), v.Address.Validate())
	}
	return errs
}
