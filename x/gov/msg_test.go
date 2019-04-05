package gov

import (
	"testing"

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
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			err := spec.Msg.Validate()
			if spec.Exp.Is(err) {
				t.Fatalf("check expected: %v  but got %+v", spec.Exp, err)
			}
		})
	}

}
