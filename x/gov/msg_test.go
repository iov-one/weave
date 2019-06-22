package gov

import (
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
)

func TestVoteMsg(t *testing.T) {
	alice := weavetest.NewCondition().Address()

	specs := map[string]struct {
		Msg VoteMsg
		Exp *errors.Error
	}{

		"Happy path": {
			Msg: VoteMsg{ProposalID: weavetest.SequenceID(1), Selected: VoteOption_Yes, Voter: alice, Metadata: &weave.Metadata{Schema: 1}},
		},
		"Voter optional": {
			Msg: VoteMsg{ProposalID: weavetest.SequenceID(1), Selected: VoteOption_Yes, Metadata: &weave.Metadata{Schema: 1}},
		},
		"Proposal id missing": {
			Msg: VoteMsg{Selected: VoteOption_Yes, Voter: alice, Metadata: &weave.Metadata{Schema: 1}},
			Exp: errors.ErrInput,
		},
		"Vote option missing": {
			Msg: VoteMsg{ProposalID: weavetest.SequenceID(1), Voter: alice, Metadata: &weave.Metadata{Schema: 1}},
			Exp: errors.ErrInput,
		},
		"Invalid vote option": {
			Msg: VoteMsg{ProposalID: weavetest.SequenceID(1), Selected: VoteOption(100), Voter: alice, Metadata: &weave.Metadata{Schema: 1}},
			Exp: errors.ErrInput,
		},
		"Invalid voter address": {
			Msg: VoteMsg{ProposalID: weavetest.SequenceID(1), Selected: VoteOption_Yes, Voter: weave.Address([]byte{0}), Metadata: &weave.Metadata{Schema: 1}},
			Exp: errors.ErrInput,
		},
		"Metadata missing": {
			Msg: VoteMsg{ProposalID: weavetest.SequenceID(1), Selected: VoteOption_Yes, Voter: alice},
			Exp: errors.ErrMetadata,
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
			Msg: TallyMsg{ProposalID: weavetest.SequenceID(1), Metadata: &weave.Metadata{Schema: 1}},
		},
		"ID missing": {
			Msg: TallyMsg{Metadata: &weave.Metadata{Schema: 1}},
			Exp: errors.ErrInput,
		},
		"Metadata missing": {
			Msg: TallyMsg{ProposalID: weavetest.SequenceID(1)},
			Exp: errors.ErrMetadata,
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

func TestCreateProposalMsg(t *testing.T) {
	alice := weavetest.NewCondition().Address()

	buildMsg := func(mods ...func(*CreateProposalMsg)) CreateProposalMsg {
		m := CreateProposalMsg{
			Metadata:       &weave.Metadata{Schema: 1},
			Title:          "any title _.-",
			Description:    "any description",
			ElectionRuleID: weavetest.SequenceID(1),
			StartTime:      weave.AsUnixTime(time.Now()),
			Author:         alice,
			RawOption:      []byte("random text, not encoded"),
		}
		for _, mod := range mods {
			mod(&m)
		}
		return m
	}

	specs := map[string]struct {
		Msg CreateProposalMsg
		Exp *errors.Error
	}{
		"Happy path": {
			Msg: buildMsg(),
		},
		"Author is optional": {
			Msg: buildMsg(func(p *CreateProposalMsg) {
				p.Author = nil
			}),
		},
		"Short title within range": {
			Msg: buildMsg(func(p *CreateProposalMsg) {
				p.Title = "fooo"
			}),
		},
		"Long title within range": {
			Msg: buildMsg(func(p *CreateProposalMsg) {
				p.Title = BigString(128)
			}),
		},
		"Title too short": {
			Msg: buildMsg(func(p *CreateProposalMsg) {
				p.Title = "foo"
			}),
			Exp: errors.ErrInput,
		},
		"Title too long": {
			Msg: buildMsg(func(p *CreateProposalMsg) {
				p.Title = BigString(129)
			}),
			Exp: errors.ErrInput,
		},
		"Title with invalid chars": {
			Msg: buildMsg(func(p *CreateProposalMsg) {
				p.Title = "title with invalid char <"
			}),
			Exp: errors.ErrInput,
		},
		"Description too short": {
			Msg: buildMsg(func(p *CreateProposalMsg) {
				p.Title = "foo"
			}),
			Exp: errors.ErrInput,
		},
		"Description too long": {
			Msg: buildMsg(func(p *CreateProposalMsg) {
				p.Title = BigString(5001)
			}),
			Exp: errors.ErrInput,
		},
		"ElectionRuleID missing": {
			Msg: buildMsg(func(p *CreateProposalMsg) {
				p.ElectionRuleID = nil
			}),
			Exp: errors.ErrInput,
		},
		"StartTime zero": {
			Msg: buildMsg(func(p *CreateProposalMsg) {
				p.StartTime = 0
			}),
			Exp: errors.ErrInput,
		},
		"Invalid author address": {
			Msg: buildMsg(func(p *CreateProposalMsg) {
				p.Author = []byte{0, 0, 0, 0}
			}),
			Exp: errors.ErrInput,
		},
		"Metadata missing": {
			Msg: buildMsg(func(p *CreateProposalMsg) {
				p.Metadata = nil
			}),
			Exp: errors.ErrMetadata,
		},
		"Options missing": {
			Msg: buildMsg(func(p *CreateProposalMsg) {
				p.RawOption = nil
			}),
			Exp: errors.ErrEmpty,
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

func TestDeleteProposalMsg(t *testing.T) {
	specs := map[string]struct {
		Msg DeleteProposalMsg
		Exp *errors.Error
	}{
		"Happy path": {
			Msg: DeleteProposalMsg{ProposalID: weavetest.SequenceID(1), Metadata: &weave.Metadata{Schema: 1}},
		},
		"Empty ID": {
			Msg: DeleteProposalMsg{Metadata: &weave.Metadata{Schema: 1}},
			Exp: errors.ErrInput,
		},
		"Metadata missing": {
			Msg: DeleteProposalMsg{ProposalID: weavetest.SequenceID(1)},
			Exp: errors.ErrMetadata,
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

func TestCreateTextResolutionMsg(t *testing.T) {
	specs := map[string]struct {
		Msg CreateTextResolutionMsg
		Exp *errors.Error
	}{
		"Happy path": {
			Msg: CreateTextResolutionMsg{Resolution: "123", Metadata: &weave.Metadata{Schema: 1}},
		},
		"Empty resolution": {
			Msg: CreateTextResolutionMsg{Metadata: &weave.Metadata{Schema: 1}},
			Exp: errors.ErrEmpty,
		},
		"Metadata missing": {
			Msg: CreateTextResolutionMsg{Resolution: "123"},
			Exp: errors.ErrMetadata,
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
