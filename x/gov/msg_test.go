package gov

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
)

func TestVoteMsg(t *testing.T) {
	specs := map[string]struct {
		Msg VoteMsg
		Exp *errors.Error
	}{

		"Happy path": {
			Msg: VoteMsg{ProposalId: weavetest.SequenceID(1), Selected: VoteOption_Yes, Voter: alice},
		},
		"Voter optional": {
			Msg: VoteMsg{ProposalId: weavetest.SequenceID(1), Selected: VoteOption_Yes},
		},
		"Proposal id missing": {
			Msg: VoteMsg{Selected: VoteOption_Yes, Voter: alice},
			Exp: errors.ErrInvalidInput,
		},
		"Vote option missing": {
			Msg: VoteMsg{ProposalId: weavetest.SequenceID(1), Voter: alice},
			Exp: errors.ErrInvalidInput,
		},
		"Invalid vote option": {
			Msg: VoteMsg{ProposalId: weavetest.SequenceID(1), Selected: VoteOption(100), Voter: alice},
			Exp: errors.ErrInvalidInput,
		},
		"Invalid voter address": {
			Msg: VoteMsg{ProposalId: weavetest.SequenceID(1), Selected: VoteOption_Yes, Voter: weave.Address([]byte{0})},
			Exp: errors.ErrInvalidInput,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			err := spec.Msg.Validate()
			if !spec.Exp.Is(err) {
				t.Fatalf("check expected: %v  but got %+v", spec.Exp, err)
			}
		})
	}

}
