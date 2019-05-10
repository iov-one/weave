package gov

import (
	"math"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/weavetest"
)

func TestElectorateValidation(t *testing.T) {
	specs := map[string]struct {
		Src Electorate
		Exp *errors.Error
	}{
		"All good with min electors count": {
			Src: Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Title:                 "My Electorate",
				Admin:                 alice,
				Electors:              []Elector{{Address: alice, Weight: 1}},
				TotalElectorateWeight: 1,
				UpdateElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
			}},
		"All good with max electors count": {
			Src: Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Title:                 "My Electorate",
				Admin:                 alice,
				Electors:              buildElectors(2000),
				TotalElectorateWeight: 2000,
				UpdateElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
			}},
		"All good with max weight count": {
			Src: Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Title:                 "My Electorate",
				Admin:                 alice,
				Electors:              []Elector{{Address: alice, Weight: 65535}},
				TotalElectorateWeight: 65535,
				UpdateElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
			}},
		"Not enough electors": {
			Src: Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Title:                 "My Electorate",
				Admin:                 alice,
				Electors:              []Elector{},
				TotalElectorateWeight: 1,
				UpdateElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
			},
			Exp: errors.ErrInput,
		},
		"Too many electors": {
			Src: Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Title:                 "My Electorate",
				Admin:                 alice,
				Electors:              buildElectors(2001),
				TotalElectorateWeight: 2001,
				UpdateElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
			},
			Exp: errors.ErrInput,
		},
		"Duplicate electors": {
			Src: Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Title:                 "My Electorate",
				Admin:                 alice,
				Electors:              []Elector{{Address: alice, Weight: 1}, {Address: alice, Weight: 1}},
				TotalElectorateWeight: 2,
				UpdateElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
			},
			Exp: errors.ErrInput,
		},
		"Empty electors weight ": {
			Src: Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Title:                 "My Electorate",
				Admin:                 alice,
				Electors:              []Elector{{Address: bobby, Weight: 0}, {Address: alice, Weight: 1}},
				TotalElectorateWeight: 1,
				UpdateElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
			},
			Exp: errors.ErrInput,
		},
		"Electors weight exceeds max": {
			Src: Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Title:                 "My Electorate",
				Admin:                 alice,
				Electors:              []Elector{{Address: alice, Weight: 65536}},
				TotalElectorateWeight: 65536,
				UpdateElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
			},
			Exp: errors.ErrInput,
		},
		"Electors address must not be empty": {
			Src: Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Title:                 "My Electorate",
				Admin:                 alice,
				Electors:              []Elector{{Address: weave.Address{}, Weight: 1}},
				TotalElectorateWeight: 1,
				UpdateElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
			},
			Exp: errors.ErrEmpty,
		},
		"Total weight mismatch": {
			Src: Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Title:                 "My Electorate",
				Admin:                 alice,
				Electors:              []Elector{{Address: alice, Weight: 1}},
				TotalElectorateWeight: 2,
				UpdateElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
			},
			Exp: errors.ErrInput,
		},
		"Title too short": {
			Src: Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Title:                 "foo",
				Admin:                 alice,
				Electors:              []Elector{{Address: alice, Weight: 1}},
				TotalElectorateWeight: 1,
				UpdateElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
			},
			Exp: errors.ErrInput,
		},
		"Title too long": {
			Src: Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Title:                 BigString(129),
				Admin:                 alice,
				Electors:              []Elector{{Address: alice, Weight: 1}},
				TotalElectorateWeight: 1,
				UpdateElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
			},
			Exp: errors.ErrInput,
		},
		"Admin must not be invalid": {
			Src: Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Title:                 "My Electorate",
				Admin:                 weave.Address{0x0, 0x1, 0x2},
				Electors:              []Elector{{Address: alice, Weight: 1}},
				TotalElectorateWeight: 1,
				UpdateElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
			},
			Exp: errors.ErrInput,
		},
		"Admin must not be empty": {
			Src: Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Title:                 "My Electorate",
				Admin:                 weave.Address{},
				Electors:              []Elector{{Address: alice, Weight: 1}},
				TotalElectorateWeight: 1,
				UpdateElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
			},
			Exp: errors.ErrEmpty,
		},
		"Update rule must not be empty": {
			Src: Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Title:                 "My Electorate",
				Admin:                 alice,
				Electors:              []Elector{{Address: alice, Weight: 1}},
				TotalElectorateWeight: 1,
			},
			Exp: errors.ErrEmpty,
		},
		"Metadata missing": {
			Src: Electorate{
				Title:                 "My Electorate",
				Admin:                 alice,
				Electors:              []Elector{{Address: alice, Weight: 1}},
				TotalElectorateWeight: 1,
			},
			Exp: errors.ErrMetadata,
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
				Metadata:          &weave.Metadata{Schema: 1},
				Title:             "My election rule",
				Admin:             alice,
				VotingPeriodHours: 1,
				Threshold:         Fraction{Numerator: 1, Denominator: 2},
			},
		},
		"Threshold fraction allowed at 0.5 ratio": {
			Src: ElectionRule{
				Metadata:          &weave.Metadata{Schema: 1},
				Title:             "My election rule",
				Admin:             alice,
				VotingPeriodHours: 1,
				Threshold:         Fraction{Numerator: 1 << 31, Denominator: math.MaxUint32},
			},
		},
		"Title too short": {
			Src: ElectionRule{
				Metadata:          &weave.Metadata{Schema: 1},
				Title:             "foo",
				Admin:             alice,
				VotingPeriodHours: 1,
				Threshold:         Fraction{Numerator: 1, Denominator: 2},
			},
			Exp: errors.ErrInput,
		},
		"Title too long": {
			Src: ElectionRule{
				Metadata:          &weave.Metadata{Schema: 1},
				Title:             BigString(129),
				Admin:             alice,
				VotingPeriodHours: 1,
				Threshold:         Fraction{Numerator: 1, Denominator: 2},
			},
			Exp: errors.ErrInput,
		},
		"Voting period empty": {
			Src: ElectionRule{
				Metadata:          &weave.Metadata{Schema: 1},
				Title:             "My election rule",
				Admin:             alice,
				VotingPeriodHours: 0,
				Threshold:         Fraction{Numerator: 1, Denominator: 2},
			},
			Exp: errors.ErrInput,
		},
		"Voting period too long": {
			Src: ElectionRule{
				Metadata:          &weave.Metadata{Schema: 1},
				Title:             "My election rule",
				Admin:             alice,
				VotingPeriodHours: 673, // = 4 * 7 * 24 + 1
				Threshold:         Fraction{Numerator: 1, Denominator: 2},
			},
			Exp: errors.ErrInput,
		},
		"Threshold must not be lower han 0.5": {
			Src: ElectionRule{
				Metadata:          &weave.Metadata{Schema: 1},
				Title:             "My election rule",
				Admin:             alice,
				VotingPeriodHours: 1,
				Threshold:         Fraction{Numerator: 1<<31 - 1, Denominator: math.MaxUint32},
			},
			Exp: errors.ErrInput,
		},
		"Threshold fraction must not be higher than 1": {
			Src: ElectionRule{
				Metadata:          &weave.Metadata{Schema: 1},
				Title:             "My election rule",
				Admin:             alice,
				VotingPeriodHours: 1,
				Threshold:         Fraction{Numerator: math.MaxUint32, Denominator: math.MaxUint32 - 1},
			},
			Exp: errors.ErrInput,
		},
		"Threshold fraction must not contain 0 numerator": {
			Src: ElectionRule{
				Metadata:          &weave.Metadata{Schema: 1},
				Title:             "My election rule",
				Admin:             alice,
				VotingPeriodHours: 1,
				Threshold:         Fraction{Numerator: 0, Denominator: math.MaxUint32 - 1},
			},
			Exp: errors.ErrInput,
		},
		"Threshold fraction must not contain 0 denominator": {
			Src: ElectionRule{
				Metadata:          &weave.Metadata{Schema: 1},
				Title:             "My election rule",
				Admin:             alice,
				VotingPeriodHours: 1,
				Threshold:         Fraction{Numerator: 1, Denominator: 0},
			},
			Exp: errors.ErrInput,
		},
		"Quorum must not be lower han 0.5": {
			Src: ElectionRule{
				Metadata:          &weave.Metadata{Schema: 1},
				Title:             "My election rule",
				Admin:             alice,
				VotingPeriodHours: 1,
				Quorum:            &Fraction{Numerator: 1<<31 - 1, Denominator: math.MaxUint32},
				Threshold:         Fraction{Numerator: 1, Denominator: 2},
			},
			Exp: errors.ErrInput,
		},
		"Quorum fraction must not be higher than 1": {
			Src: ElectionRule{
				Metadata:          &weave.Metadata{Schema: 1},
				Title:             "My election rule",
				Admin:             alice,
				VotingPeriodHours: 1,
				Quorum:            &Fraction{Numerator: math.MaxUint32, Denominator: math.MaxUint32 - 1},
				Threshold:         Fraction{Numerator: 1, Denominator: 2},
			},
			Exp: errors.ErrInput,
		},
		"Quorum fraction must not contain 0 numerator": {
			Src: ElectionRule{
				Metadata:          &weave.Metadata{Schema: 1},
				Title:             "My election rule",
				Admin:             alice,
				VotingPeriodHours: 1,
				Quorum:            &Fraction{Numerator: 0, Denominator: math.MaxUint32 - 1},
				Threshold:         Fraction{Numerator: 1, Denominator: 2},
			},
			Exp: errors.ErrInput,
		},
		"Quorum fraction must not contain 0 denominator": {
			Src: ElectionRule{
				Metadata:          &weave.Metadata{Schema: 1},
				Title:             "My election rule",
				Admin:             alice,
				VotingPeriodHours: 1,
				Quorum:            &Fraction{Numerator: 1, Denominator: 0},
				Threshold:         Fraction{Numerator: 1, Denominator: 2},
			},
			Exp: errors.ErrInput,
		},
		"Admin must not be invalid": {
			Src: ElectionRule{
				Metadata:          &weave.Metadata{Schema: 1},
				Title:             "My election rule",
				Admin:             weave.Address{0x0, 0x1, 0x2},
				VotingPeriodHours: 1,
				Quorum:            &Fraction{Numerator: 1, Denominator: 1},
				Threshold:         Fraction{Numerator: 1, Denominator: 2},
			},
			Exp: errors.ErrInput,
		},
		"Admin must not be empty": {
			Src: ElectionRule{
				Metadata:          &weave.Metadata{Schema: 1},
				Title:             "My election rule",
				Admin:             weave.Address{},
				VotingPeriodHours: 1,
				Quorum:            &Fraction{Numerator: 1, Denominator: 1},
				Threshold:         Fraction{Numerator: 1, Denominator: 2},
			},
			Exp: errors.ErrEmpty,
		},
		"Missing metadata": {
			Src: ElectionRule{
				Title:             "My election rule",
				Admin:             alice,
				VotingPeriodHours: 1,
				Threshold:         Fraction{Numerator: 1, Denominator: 2},
			},
			Exp: errors.ErrMetadata,
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
		Src Proposal
		Exp *errors.Error
	}{
		"Happy path": {
			Src: textProposalFixture(),
		},
		"Title too short": {
			Src: textProposalFixture(func(p *Proposal) {
				p.Title = "foo"
			}),
			Exp: errors.ErrInput,
		},
		"Title too long": {
			Src: textProposalFixture(func(p *Proposal) {
				p.Title = BigString(129)
			}),
			Exp: errors.ErrInput,
		},
		"Description empty": {
			Src: textProposalFixture(func(p *Proposal) {
				p.Description = ""
			}),
			Exp: errors.ErrInput,
		},
		"Description too long": {
			Src: textProposalFixture(func(p *Proposal) {
				p.Description = BigString(5001)
			}),
			Exp: errors.ErrInput,
		},
		"Author missing": {
			Src: textProposalFixture(func(p *Proposal) {
				p.Author = nil
			}),
			Exp: errors.ErrInput,
		},
		"ElectorateRef invalid": {
			Src: textProposalFixture(func(p *Proposal) {
				p.ElectorateRef = orm.VersionedIDRef{}
			}),
			Exp: errors.ErrEmpty,
		},
		"ElectionRuleID missing": {
			Src: textProposalFixture(func(p *Proposal) {
				p.ElectionRuleRef = orm.VersionedIDRef{}
			}),
			Exp: errors.ErrEmpty,
		},
		"StartTime missing": {
			Src: textProposalFixture(func(p *Proposal) {
				var unset time.Time
				p.VotingStartTime = weave.AsUnixTime(unset)
			}),
			Exp: errors.ErrInput,
		},
		"EndTime missing": {
			Src: textProposalFixture(func(p *Proposal) {
				var unset time.Time
				p.VotingEndTime = weave.AsUnixTime(unset)
			}),
			Exp: errors.ErrInput,
		},
		"Status missing": {
			Src: textProposalFixture(func(p *Proposal) {
				p.Status = Proposal_Status(0)
			}),
			Exp: errors.ErrInput,
		},
		"Result missing": {
			Src: textProposalFixture(func(p *Proposal) {
				p.Result = Proposal_Result(0)
			}),
			Exp: errors.ErrInput,
		},
		"Metadata missing": {
			Src: textProposalFixture(func(p *Proposal) {
				p.Metadata = nil
			}),
			Exp: errors.ErrMetadata,
		},
		"Details missing": {
			Src: updateElectorateProposalFixture(func(p *Proposal) {
				p.Details = nil
			}),
			Exp: errors.ErrEmpty,
		},
		"Electorate diff missing": {
			Src: updateElectorateProposalFixture(func(p *Proposal) {
				p.GetElectorateUpdateDetails().DiffElectors = nil
			}),
			Exp: errors.ErrEmpty,
		},
		"Electorate invalid ": {
			Src: updateElectorateProposalFixture(func(p *Proposal) {
				p.GetElectorateUpdateDetails().DiffElectors = []Elector{{Address: alice, Weight: math.MaxUint32}}
			}),
			Exp: errors.ErrInput,
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
				Metadata: &weave.Metadata{Schema: 1},
				Voted:    VoteOption_Yes,
				Elector:  Elector{Address: bobby, Weight: 10},
			},
		},
		"Voted option missing": {
			Src: Vote{Elector: Elector{Address: bobby, Weight: 10}, Metadata: &weave.Metadata{Schema: 1}},
			Exp: errors.ErrInput,
		},
		"Elector missing": {
			Src: Vote{Voted: VoteOption_Yes, Metadata: &weave.Metadata{Schema: 1}},
			Exp: errors.ErrInput,
		},
		"Elector's weight missing": {
			Src: Vote{Voted: VoteOption_Yes, Elector: Elector{Address: bobby}, Metadata: &weave.Metadata{Schema: 1}},
			Exp: errors.ErrInput,
		},
		"Elector's Address missing": {
			Src: Vote{Voted: VoteOption_Yes, Elector: Elector{Weight: 1}, Metadata: &weave.Metadata{Schema: 1}},
			Exp: errors.ErrEmpty,
		},
		"Invalid option": {
			Src: Vote{Voted: VoteOption_Invalid, Elector: Elector{Address: bobby, Weight: 1}, Metadata: &weave.Metadata{Schema: 1}},
			Exp: errors.ErrInput,
		},
		"Metadata missing": {
			Src: Vote{
				Voted:   VoteOption_Yes,
				Elector: Elector{Address: bobby, Weight: 10},
			},
			Exp: errors.ErrMetadata,
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

func textProposalFixture(mods ...func(*Proposal)) Proposal {
	now := weave.AsUnixTime(time.Now())
	proposal := Proposal{
		Metadata:        &weave.Metadata{Schema: 1},
		Type:            Proposal_Text,
		Title:           "My proposal",
		Description:     "My description",
		ElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
		ElectorateRef:   orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
		VotingStartTime: now.Add(-1 * time.Minute),
		VotingEndTime:   now.Add(time.Minute),
		SubmissionTime:  now.Add(-1 * time.Hour),
		Status:          Proposal_Submitted,
		Result:          Proposal_Undefined,
		Author:          alice,
		VoteState:       NewTallyResult(nil, Fraction{1, 2}, 11),
		Details:         &Proposal_TextDetails{&TextProposalPayload{}},
	}
	for _, mod := range mods {
		if mod != nil {
			mod(&proposal)
		}
	}
	return proposal
}

func updateElectorateProposalFixture(mods ...func(*Proposal)) Proposal {
	now := weave.AsUnixTime(time.Now())
	proposal := Proposal{
		Metadata:        &weave.Metadata{Schema: 1},
		Type:            Proposal_UpdateElectorate,
		Title:           "My proposal",
		Description:     "My description",
		ElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
		ElectorateRef:   orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
		VotingStartTime: now.Add(-1 * time.Minute),
		VotingEndTime:   now.Add(time.Minute),
		SubmissionTime:  now.Add(-1 * time.Hour),
		Status:          Proposal_Submitted,
		Result:          Proposal_Undefined,
		Author:          alice,
		VoteState:       NewTallyResult(nil, Fraction{1, 2}, 11),
		Details: &Proposal_ElectorateUpdateDetails{&ElectorateUpdatePayload{
			[]Elector{{Address: alice, Weight: 10}},
		}},
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
		r[i] = Elector{Weight: 1, Address: weavetest.NewCondition().Address()}
	}
	return r
}
