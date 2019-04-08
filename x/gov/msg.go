package gov

import (
	"fmt"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

const (
	pathCreateTextProposalMsg = "gov/create"
	pathVoteMsg               = "gov/vote"
	pathTallyMsg              = "gov/tally"
)

var _ weave.Msg = (*CreateTextProposalMsg)(nil)
var _ weave.Msg = (*VoteMsg)(nil)
var _ weave.Msg = (*TallyMsg)(nil)

func (CreateTextProposalMsg) Path() string {
	return pathCreateTextProposalMsg
}

func (m CreateTextProposalMsg) Validate() error {
	err := m.Author.Validate()
	switch {
	case len(m.ElectorateId) == 0:
		return errors.Wrap(errors.ErrInvalidInput, "empty electorate id")
	case len(m.ElectionRuleId) == 0:
		return errors.Wrap(errors.ErrInvalidInput, "empty election rules id")
	case m.StartTime.Time().IsZero():
		return errors.Wrap(errors.ErrInvalidInput, "empty start time")
	case m.Author != nil && err != nil:
		return errors.Wrap(err, "invalid author")
	case !validTitle(m.Title):
		return errors.Wrap(errors.ErrInvalidInput, fmt.Sprintf("title: %q", m.Title))
	case len(m.Description) < minDescriptionLength:
		return errors.Wrap(errors.ErrInvalidInput, fmt.Sprintf("description length lower than minimum of: %d", minDescriptionLength))
	case len(m.Description) > maxDescriptionLength:
		return errors.Wrap(errors.ErrInvalidInput, fmt.Sprintf("description length exceeds: %d", maxDescriptionLength))
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
	if len(m.ProposalId) == 0 {
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
	if len(m.ProposalId) == 0 {
		return errors.Wrap(errors.ErrInvalidInput, "empty proposal id")
	}
	return nil
}
