package gov

import (
	"testing"
	"time"

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
			Msg: VoteMsg{ProposalID: weavetest.SequenceID(1), Selected: VoteOption_Yes, Voter: alice},
		},
		"Voter optional": {
			Msg: VoteMsg{ProposalID: weavetest.SequenceID(1), Selected: VoteOption_Yes},
		},
		"Proposal id missing": {
			Msg: VoteMsg{Selected: VoteOption_Yes, Voter: alice},
			Exp: errors.ErrInvalidInput,
		},
		"Vote option missing": {
			Msg: VoteMsg{ProposalID: weavetest.SequenceID(1), Voter: alice},
			Exp: errors.ErrInvalidInput,
		},
		"Invalid vote option": {
			Msg: VoteMsg{ProposalID: weavetest.SequenceID(1), Selected: VoteOption(100), Voter: alice},
			Exp: errors.ErrInvalidInput,
		},
		"Invalid voter address": {
			Msg: VoteMsg{ProposalID: weavetest.SequenceID(1), Selected: VoteOption_Yes, Voter: weave.Address([]byte{0})},
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

func TestTallyMsg(t *testing.T) {
	specs := map[string]struct {
		Msg TallyMsg
		Exp *errors.Error
	}{
		"Happy path": {
			Msg: TallyMsg{ProposalID: weavetest.SequenceID(1)},
		},
		"ID missing": {
			Msg: TallyMsg{},
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

func TestCrateTextProposalMsg(t *testing.T) {
	buildMsg := func(mods ...func(*CreateTextProposalMsg)) CreateTextProposalMsg {
		m := CreateTextProposalMsg{
			Title:          "any title _.-",
			Description:    "any description",
			ElectorateID:   weavetest.SequenceID(1),
			ElectionRuleID: weavetest.SequenceID(1),
			StartTime:      weave.AsUnixTime(time.Now()),
			Author:         alice,
		}
		for _, mod := range mods {
			mod(&m)
		}
		return m
	}

	specs := map[string]struct {
		Msg CreateTextProposalMsg
		Exp *errors.Error
	}{
		"Happy path": {
			Msg: buildMsg(),
		},
		"Author is optional": {
			Msg: buildMsg(func(p *CreateTextProposalMsg) {
				p.Author = nil
			}),
		},
		"Short title within range": {
			Msg: buildMsg(func(p *CreateTextProposalMsg) {
				p.Title = "fooo"
			}),
		},
		"Long title within range": {
			Msg: buildMsg(func(p *CreateTextProposalMsg) {
				p.Title = BigString(128)
			}),
		},
		"Title too short": {
			Msg: buildMsg(func(p *CreateTextProposalMsg) {
				p.Title = "foo"
			}),
			Exp: errors.ErrInvalidInput,
		},
		"Title too long": {
			Msg: buildMsg(func(p *CreateTextProposalMsg) {
				p.Title = BigString(129)
			}),
			Exp: errors.ErrInvalidInput,
		},
		"Title with invalid chars": {
			Msg: buildMsg(func(p *CreateTextProposalMsg) {
				p.Title = "title with invalid char <"
			}),
			Exp: errors.ErrInvalidInput,
		},
		"Description too short": {
			Msg: buildMsg(func(p *CreateTextProposalMsg) {
				p.Title = "foo"
			}),
			Exp: errors.ErrInvalidInput,
		},
		"Description too long": {
			Msg: buildMsg(func(p *CreateTextProposalMsg) {
				p.Title = BigString(5001)
			}),
			Exp: errors.ErrInvalidInput,
		},
		"ElectorateID missing": {
			Msg: buildMsg(func(p *CreateTextProposalMsg) {
				p.ElectorateID = nil
			}),
			Exp: errors.ErrInvalidInput,
		},
		"ElectionRuleID missing": {
			Msg: buildMsg(func(p *CreateTextProposalMsg) {
				p.ElectionRuleID = nil
			}),
			Exp: errors.ErrInvalidInput,
		},
		"StartTime zero": {
			Msg: buildMsg(func(p *CreateTextProposalMsg) {
				p.StartTime = 0
			}),
			Exp: errors.ErrInvalidInput,
		},
		"Invalid author address": {
			Msg: buildMsg(func(p *CreateTextProposalMsg) {
				p.Author = []byte{0, 0, 0, 0}
			}),
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

func BigString(n int) string {
	const randomChar = "a"
	var r string
	for i := 0; i < n; i++ {
		r += randomChar
	}
	return r
}
