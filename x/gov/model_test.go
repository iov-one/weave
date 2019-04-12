package gov

import (
	"math"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
)

func TestElectorateValidation(t *testing.T) {
	specs := map[string]struct {
		Src Electorate
		Exp *errors.Error
	}{
		"All good with min electors count": {
			Src: Electorate{
				Title:                 "My Electorate",
				Electors:              []Elector{{Signature: alice, Weight: 1}},
				TotalWeightElectorate: 1,
			}},
		"All good with max electors count": {
			Src: Electorate{
				Title:                 "My Electorate",
				Electors:              buildElectors(2000),
				TotalWeightElectorate: 2000,
			}},
		"Not enough electors": {
			Src: Electorate{
				Title:                 "My Electorate",
				Electors:              []Elector{},
				TotalWeightElectorate: 1,
			},
			Exp: errors.ErrInvalidInput,
		},
		"Too many electors": {
			Src: Electorate{
				Title:                 "My Electorate",
				Electors:              buildElectors(2001),
				TotalWeightElectorate: 2001,
			},
			Exp: errors.ErrInvalidInput,
		},
		"Duplicate electors": {
			Src: Electorate{
				Title:                 "My Electorate",
				Electors:              []Elector{{Signature: alice, Weight: 1}, {Signature: alice, Weight: 1}},
				TotalWeightElectorate: 2,
			},
			Exp: errors.ErrInvalidInput,
		},
		"Empty electors weight ": {
			Src: Electorate{
				Title:                 "My Electorate",
				Electors:              []Elector{{Signature: bobby, Weight: 0}, {Signature: alice, Weight: 1}},
				TotalWeightElectorate: 1,
			},
			Exp: errors.ErrInvalidInput,
		},
		"Electors weight exceeds max": {
			Src: Electorate{
				Title:                 "My Electorate",
				Electors:              []Elector{{Signature: alice, Weight: 65536}},
				TotalWeightElectorate: 65536,
			},
			Exp: errors.ErrInvalidInput,
		},
		"Total weight mismatch": {
			Src: Electorate{
				Title:                 "My Electorate",
				Electors:              []Elector{{Signature: alice, Weight: 1}},
				TotalWeightElectorate: 2,
			},
			Exp: errors.ErrInvalidInput,
		},
		"Title too short": {
			Src: Electorate{
				Title:                 "foo",
				Electors:              []Elector{{Signature: alice, Weight: 1}},
				TotalWeightElectorate: 1,
			},
			Exp: errors.ErrInvalidInput,
		},
		"Title too long": {
			Src: Electorate{
				Title:                 BigString(129),
				Electors:              []Elector{{Signature: alice, Weight: 1}},
				TotalWeightElectorate: 1,
			},
			Exp: errors.ErrInvalidInput,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			if exp, got := spec.Exp, spec.Src.Validate(); !exp.Is(got) {
				t.Errorf("expected %v but got %v", exp, got)
			}
		})
	}
}

func TestElectionRuleValidation(t *testing.T) {
	specs := map[string]struct {
		Src ElectionRule
		Exp *errors.Error
	}{
		"All good": {
			Src: ElectionRule{
				Title:             "My election rule",
				VotingPeriodHours: 1,
				Threshold:         Fraction{Numerator: 1, Denominator: 2},
			},
		},
		"Threshold fraction allowed at 0.5 ratio": {
			Src: ElectionRule{
				Title:             "My election rule",
				VotingPeriodHours: 1,
				Threshold:         Fraction{Numerator: 1 << 31, Denominator: math.MaxUint32},
			},
		},
		"Title too short": {
			Src: ElectionRule{
				Title:             "foo",
				VotingPeriodHours: 1,
				Threshold:         Fraction{Numerator: 1, Denominator: 2},
			},
			Exp: errors.ErrInvalidInput,
		},
		"Title too long": {
			Src: ElectionRule{
				Title:             BigString(129),
				VotingPeriodHours: 1,
				Threshold:         Fraction{Numerator: 1, Denominator: 2},
			},
			Exp: errors.ErrInvalidInput,
		},
		"Voting period empty": {
			Src: ElectionRule{
				Title:             "My election rule",
				VotingPeriodHours: 0,
				Threshold:         Fraction{Numerator: 1, Denominator: 2},
			},
			Exp: errors.ErrInvalidInput,
		},
		"Voting period too long": {
			Src: ElectionRule{
				Title:             "My election rule",
				VotingPeriodHours: 673, // = 4 * 7 * 24 + 1
				Threshold:         Fraction{Numerator: 1, Denominator: 2},
			},
			Exp: errors.ErrInvalidInput,
		},
		"Threshold must not be lower han 0.5": {
			Src: ElectionRule{
				Title:             "My election rule",
				VotingPeriodHours: 1,
				Threshold:         Fraction{Numerator: 1<<31 - 1, Denominator: math.MaxUint32},
			},
			Exp: errors.ErrInvalidInput,
		},
		"Threshold fraction must not be higher than 1": {
			Src: ElectionRule{
				Title:             "My election rule",
				VotingPeriodHours: 1,
				Threshold:         Fraction{Numerator: math.MaxUint32, Denominator: math.MaxUint32 - 1},
			},
			Exp: errors.ErrInvalidInput,
		},
		"Threshold fraction must not contain 0 numerator": {
			Src: ElectionRule{
				Title:             "My election rule",
				VotingPeriodHours: 1,
				Threshold:         Fraction{Numerator: 0, Denominator: math.MaxUint32 - 1},
			},
			Exp: errors.ErrInvalidInput,
		},
		"Threshold fraction must not contain 0 denominator": {
			Src: ElectionRule{
				Title:             "My election rule",
				VotingPeriodHours: 1,
				Threshold:         Fraction{Numerator: 1, Denominator: 0},
			},
			Exp: errors.ErrInvalidInput,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			if exp, got := spec.Exp, spec.Src.Validate(); !exp.Is(got) {
				t.Errorf("expected %v but got %v", exp, got)
			}
		})
	}
}

func TestTextProposalValidation(t *testing.T) {
	specs := map[string]struct {
		Src TextProposal
		Exp *errors.Error
	}{
		"Happy path": {
			Src: textProposalFixture(),
		},
		"Title too short": {
			Src: textProposalFixture(func(p *TextProposal) {
				p.Title = "foo"
			}),
			Exp: errors.ErrInvalidInput,
		},
		"Title too long": {
			Src: textProposalFixture(func(p *TextProposal) {
				p.Title = BigString(129)
			}),
			Exp: errors.ErrInvalidInput,
		},
		"Description empty": {
			Src: textProposalFixture(func(p *TextProposal) {
				p.Description = ""
			}),
			Exp: errors.ErrInvalidInput,
		},
		"Description too long": {
			Src: textProposalFixture(func(p *TextProposal) {
				p.Description = BigString(5001)
			}),
			Exp: errors.ErrInvalidInput,
		},
		"Author missing": {
			Src: textProposalFixture(func(p *TextProposal) {
				p.Author = nil
			}),
			Exp: errors.ErrInvalidInput,
		},
		"ElectorateID missing": {
			Src: textProposalFixture(func(p *TextProposal) {
				p.ElectorateID = nil
			}),
			Exp: errors.ErrInvalidInput,
		},
		"ElectionRuleID missing": {
			Src: textProposalFixture(func(p *TextProposal) {
				p.ElectionRuleID = nil
			}),
			Exp: errors.ErrInvalidInput,
		},
		"StartTime missing": {
			Src: textProposalFixture(func(p *TextProposal) {
				var unset time.Time
				p.VotingStartTime = weave.AsUnixTime(unset)
			}),
			Exp: errors.ErrInvalidInput,
		},
		"EndTime missing": {
			Src: textProposalFixture(func(p *TextProposal) {
				var unset time.Time
				p.VotingEndTime = weave.AsUnixTime(unset)
			}),
			Exp: errors.ErrInvalidInput,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			if exp, got := spec.Exp, spec.Src.Validate(); !exp.Is(got) {
				t.Errorf("expected %v but got %v", exp, got)
			}
		})
	}
}

func TestVoteValidate(t *testing.T) {
	specs := map[string]struct {
		Src Vote
		Exp *errors.Error
	}{
		"All good": {
			Src: Vote{
				Voted:   VoteOption_Yes,
				Elector: Elector{Signature: bobby, Weight: 10},
			},
		},
		"Voted option missing": {
			Src: Vote{Elector: Elector{Signature: bobby, Weight: 10}},
			Exp: errors.ErrInvalidInput,
		},
		"Elector missing": {
			Src: Vote{Voted: VoteOption_Yes},
			Exp: errors.ErrInvalidInput,
		},
		"Elector's weight missing": {
			Src: Vote{Voted: VoteOption_Yes, Elector: Elector{Signature: bobby}},
			Exp: errors.ErrInvalidInput,
		},
		"Elector's signature missing": {
			Src: Vote{Voted: VoteOption_Yes, Elector: Elector{Weight: 1}},
			Exp: errors.ErrInvalidInput,
		},
		"Invalid option": {
			Src: Vote{Voted: VoteOption_Invalid, Elector: Elector{Signature: bobby, Weight: 1}},
			Exp: errors.ErrInvalidInput,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			if exp, got := spec.Exp, spec.Src.Validate(); !exp.Is(got) {
				t.Errorf("expected %v but got %v", exp, got)
			}
		})
	}
}

func textProposalFixture(mods ...func(*TextProposal)) TextProposal {
	now := weave.AsUnixTime(time.Now())
	proposal := TextProposal{
		Title:           "My proposal",
		Description:     "My description",
		ElectionRuleID:  weavetest.SequenceID(1),
		ElectorateID:    weavetest.SequenceID(1),
		VotingStartTime: now.Add(-1 * time.Minute),
		VotingEndTime:   now.Add(time.Minute),
		SubmissionTime:  now.Add(-1 * time.Hour),
		Status:          TextProposal_Undefined,
		Author:          alice,
	}
	for _, mod := range mods {
		if mod != nil {
			mod(&proposal)
		}
	}
	return proposal
}

func buildElectors(n int) []Elector {
	r := make([]Elector, n)
	for i := 0; i < n; i++ {
		r[i] = Elector{Weight: 1, Signature: weavetest.NewCondition().Address()}
	}
	return r
}
