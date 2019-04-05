package gov

import (
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
	return nil
}
