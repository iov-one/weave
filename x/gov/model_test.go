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
	alice := weavetest.NewCondition().Address()
	bobby := weavetest.NewCondition().Address()

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
			}},
		"All good with max electors count": {
			Src: Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Title:                 "My Electorate",
				Admin:                 alice,
				Electors:              buildElectors(2000),
				TotalElectorateWeight: 2000,
			}},
		"All good with max weight count": {
			Src: Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Title:                 "My Electorate",
				Admin:                 alice,
				Electors:              []Elector{{Address: alice, Weight: 65535}},
				TotalElectorateWeight: 65535,
			}},
		"Not enough electors": {
			Src: Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Title:                 "My Electorate",
				Admin:                 alice,
				Electors:              []Elector{},
				TotalElectorateWeight: 1,
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
	alice := weavetest.NewCondition().Address()

	specs := map[string]struct {
		Src ElectionRule
		Exp *errors.Error
	}{
		"All good": {
			Src: ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "My election rule",
				Admin:        alice,
				VotingPeriod: weave.AsUnixDuration(time.Hour),
				Threshold:    Fraction{Numerator: 1, Denominator: 2},
				ElectorateID: weavetest.SequenceID(5),
				Address:      Condition(weavetest.SequenceID(6)).Address(),
			},
		},
		"Address should be valid": {
			Src: ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "My election rule",
				Admin:        alice,
				VotingPeriod: weave.AsUnixDuration(time.Hour),
				Threshold:    Fraction{Numerator: 1, Denominator: 2},
				ElectorateID: weavetest.SequenceID(5),
			},
			Exp: errors.ErrEmpty,
		},
		"ElectorateID must be present": {
			Src: ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "My election rule",
				Admin:        alice,
				VotingPeriod: weave.AsUnixDuration(time.Hour),
				Threshold:    Fraction{Numerator: 1, Denominator: 2},
				Address:      Condition(weavetest.SequenceID(6)).Address(),
			},
			Exp: errors.ErrEmpty,
		},
		"ElectorateID must be 8 bytes": {
			Src: ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "My election rule",
				Admin:        alice,
				VotingPeriod: weave.AsUnixDuration(time.Hour),
				Threshold:    Fraction{Numerator: 1, Denominator: 2},
				ElectorateID: []byte("foo"),
				Address:      Condition(weavetest.SequenceID(6)).Address(),
			},
			Exp: errors.ErrInput,
		},
		"Threshold fraction allowed at 0.5 ratio": {
			Src: ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "My election rule",
				Admin:        alice,
				VotingPeriod: weave.AsUnixDuration(time.Hour),
				Threshold:    Fraction{Numerator: 1 << 31, Denominator: math.MaxUint32},
				ElectorateID: weavetest.SequenceID(5),
				Address:      Condition(weavetest.SequenceID(6)).Address(),
			},
		},
		"Title too short": {
			Src: ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "foo",
				Admin:        alice,
				VotingPeriod: weave.AsUnixDuration(time.Hour),
				Threshold:    Fraction{Numerator: 1, Denominator: 2},
				ElectorateID: weavetest.SequenceID(5),
				Address:      Condition(weavetest.SequenceID(6)).Address(),
			},
			Exp: errors.ErrInput,
		},
		"Title too long": {
			Src: ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        BigString(129),
				Admin:        alice,
				VotingPeriod: weave.AsUnixDuration(time.Hour),
				Threshold:    Fraction{Numerator: 1, Denominator: 2},
				ElectorateID: weavetest.SequenceID(5),
				Address:      Condition(weavetest.SequenceID(6)).Address(),
			},
			Exp: errors.ErrInput,
		},
		"Voting period empty": {
			Src: ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "My election rule",
				Admin:        alice,
				VotingPeriod: weave.AsUnixDuration(0),
				Threshold:    Fraction{Numerator: 1, Denominator: 2},
				ElectorateID: weavetest.SequenceID(5),
				Address:      Condition(weavetest.SequenceID(6)).Address(),
			},
			Exp: errors.ErrInput,
		},
		"Voting period too long": {
			Src: ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "My election rule",
				Admin:        alice,
				VotingPeriod: weave.AsUnixDuration((24*7*4 + 1) * time.Hour),
				Threshold:    Fraction{Numerator: 1, Denominator: 2},
				ElectorateID: weavetest.SequenceID(5),
				Address:      Condition(weavetest.SequenceID(6)).Address(),
			},
			Exp: errors.ErrInput,
		},
		"Threshold must not be lower han 0.5": {
			Src: ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "My election rule",
				Admin:        alice,
				VotingPeriod: weave.AsUnixDuration(time.Hour),
				Threshold:    Fraction{Numerator: 1<<31 - 1, Denominator: math.MaxUint32},
				ElectorateID: weavetest.SequenceID(5),
				Address:      Condition(weavetest.SequenceID(6)).Address(),
			},
			Exp: errors.ErrInput,
		},
		"Threshold fraction must not be higher than 1": {
			Src: ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "My election rule",
				Admin:        alice,
				VotingPeriod: weave.AsUnixDuration(time.Hour),
				Threshold:    Fraction{Numerator: math.MaxUint32, Denominator: math.MaxUint32 - 1},
				ElectorateID: weavetest.SequenceID(5),
				Address:      Condition(weavetest.SequenceID(6)).Address(),
			},
			Exp: errors.ErrInput,
		},
		"Threshold fraction must not contain 0 numerator": {
			Src: ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "My election rule",
				Admin:        alice,
				VotingPeriod: weave.AsUnixDuration(time.Hour),
				Threshold:    Fraction{Numerator: 0, Denominator: math.MaxUint32 - 1},
				ElectorateID: weavetest.SequenceID(5),
				Address:      Condition(weavetest.SequenceID(6)).Address(),
			},
			Exp: errors.ErrInput,
		},
		"Threshold fraction must not contain 0 denominator": {
			Src: ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "My election rule",
				Admin:        alice,
				VotingPeriod: weave.AsUnixDuration(time.Hour),
				Threshold:    Fraction{Numerator: 1, Denominator: 0},
				ElectorateID: weavetest.SequenceID(5),
				Address:      Condition(weavetest.SequenceID(6)).Address(),
			},
			Exp: errors.ErrInput,
		},
		"Quorum must not be lower han 0.5": {
			Src: ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "My election rule",
				Admin:        alice,
				VotingPeriod: weave.AsUnixDuration(time.Hour),
				Quorum:       &Fraction{Numerator: 1<<31 - 1, Denominator: math.MaxUint32},
				Threshold:    Fraction{Numerator: 1, Denominator: 2},
				ElectorateID: weavetest.SequenceID(5),
				Address:      Condition(weavetest.SequenceID(6)).Address(),
			},
			Exp: errors.ErrInput,
		},
		"Quorum fraction must not be higher than 1": {
			Src: ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "My election rule",
				Admin:        alice,
				VotingPeriod: weave.AsUnixDuration(time.Hour),
				Quorum:       &Fraction{Numerator: math.MaxUint32, Denominator: math.MaxUint32 - 1},
				Threshold:    Fraction{Numerator: 1, Denominator: 2},
				ElectorateID: weavetest.SequenceID(5),
				Address:      Condition(weavetest.SequenceID(6)).Address(),
			},
			Exp: errors.ErrInput,
		},
		"Quorum fraction must not contain 0 numerator": {
			Src: ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "My election rule",
				Admin:        alice,
				VotingPeriod: weave.AsUnixDuration(time.Hour),
				Quorum:       &Fraction{Numerator: 0, Denominator: math.MaxUint32 - 1},
				Threshold:    Fraction{Numerator: 1, Denominator: 2},
				ElectorateID: weavetest.SequenceID(5),
				Address:      Condition(weavetest.SequenceID(6)).Address(),
			},
			Exp: errors.ErrInput,
		},
		"Quorum fraction must not contain 0 denominator": {
			Src: ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "My election rule",
				Admin:        alice,
				VotingPeriod: weave.AsUnixDuration(time.Hour),
				Quorum:       &Fraction{Numerator: 1, Denominator: 0},
				Threshold:    Fraction{Numerator: 1, Denominator: 2},
				ElectorateID: weavetest.SequenceID(5),
				Address:      Condition(weavetest.SequenceID(6)).Address(),
			},
			Exp: errors.ErrInput,
		},
		"Admin must not be invalid": {
			Src: ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "My election rule",
				Admin:        weave.Address{0x0, 0x1, 0x2},
				VotingPeriod: weave.AsUnixDuration(time.Hour),
				Quorum:       &Fraction{Numerator: 1, Denominator: 1},
				Threshold:    Fraction{Numerator: 1, Denominator: 2},
				ElectorateID: weavetest.SequenceID(5),
				Address:      Condition(weavetest.SequenceID(6)).Address(),
			},
			Exp: errors.ErrInput,
		},
		"Admin must not be empty": {
			Src: ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "My election rule",
				Admin:        weave.Address{},
				VotingPeriod: weave.AsUnixDuration(time.Hour),
				Quorum:       &Fraction{Numerator: 1, Denominator: 1},
				Threshold:    Fraction{Numerator: 1, Denominator: 2},
				ElectorateID: weavetest.SequenceID(5),
				Address:      Condition(weavetest.SequenceID(6)).Address(),
			},
			Exp: errors.ErrEmpty,
		},
		"Missing metadata": {
			Src: ElectionRule{
				Title:        "My election rule",
				Admin:        alice,
				VotingPeriod: weave.AsUnixDuration(time.Hour),
				Threshold:    Fraction{Numerator: 1, Denominator: 2},
				ElectorateID: weavetest.SequenceID(5),
				Address:      Condition(weavetest.SequenceID(6)).Address(),
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

func TestProposalValidation(t *testing.T) {
	alice := weavetest.NewCondition().Address()

	specs := map[string]struct {
		Src Proposal
		Exp *errors.Error
	}{
		"Happy path": {
			Src: proposalFixture(t, alice),
		},
		"Title too short": {
			Src: proposalFixture(t, alice, func(p *Proposal) {
				p.Title = "foo"
			}),
			Exp: errors.ErrState,
		},
		"Title too long": {
			Src: proposalFixture(t, alice, func(p *Proposal) {
				p.Title = BigString(129)
			}),
			Exp: errors.ErrState,
		},
		"Description empty": {
			Src: proposalFixture(t, alice, func(p *Proposal) {
				p.Description = ""
			}),
			Exp: errors.ErrState,
		},
		"Description too long": {
			Src: proposalFixture(t, alice, func(p *Proposal) {
				p.Description = BigString(5001)
			}),
			Exp: errors.ErrState,
		},
		"Author missing": {
			Src: proposalFixture(t, nil),
			Exp: errors.ErrState,
		},
		"ElectorateRef invalid": {
			Src: proposalFixture(t, alice, func(p *Proposal) {
				p.ElectorateRef = orm.VersionedIDRef{}
			}),
			Exp: errors.ErrEmpty,
		},
		"ElectionRuleID missing": {
			Src: proposalFixture(t, alice, func(p *Proposal) {
				p.ElectionRuleRef = orm.VersionedIDRef{}
			}),
			Exp: errors.ErrEmpty,
		},
		"StartTime missing": {
			Src: proposalFixture(t, alice, func(p *Proposal) {
				var unset time.Time
				p.VotingStartTime = weave.AsUnixTime(unset)
			}),
			Exp: errors.ErrState,
		},
		"EndTime missing": {
			Src: proposalFixture(t, alice, func(p *Proposal) {
				var unset time.Time
				p.VotingEndTime = weave.AsUnixTime(unset)
			}),
			Exp: errors.ErrState,
		},
		"Status missing": {
			Src: proposalFixture(t, alice, func(p *Proposal) {
				p.Status = Proposal_Status(0)
			}),
			Exp: errors.ErrState,
		},
		"Result missing": {
			Src: proposalFixture(t, alice, func(p *Proposal) {
				p.Result = Proposal_Result(0)
			}),
			Exp: errors.ErrState,
		},
		"Metadata missing": {
			Src: proposalFixture(t, alice, func(p *Proposal) {
				p.Metadata = nil
			}),
			Exp: errors.ErrMetadata,
		},
		"Options missing": {
			Src: proposalFixture(t, alice, func(p *Proposal) {
				p.RawOption = nil
			}),
			Exp: errors.ErrState,
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
	bobby := weavetest.NewCondition().Address()

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

func TestResolutionValidate(t *testing.T) {
	specs := map[string]struct {
		Mutator func(r *Resolution)
		Exp     *errors.Error
	}{
		"Happy path": {},
		"Empty resolution": {
			Mutator: func(r *Resolution) {
				r.Resolution = ""
			},
			Exp: errors.ErrEmpty,
		},
		"Metadata missing": {
			Mutator: func(r *Resolution) {
				r.Metadata = nil
			},
			Exp: errors.ErrMetadata,
		},
		"ProposalID missing": {
			Mutator: func(r *Resolution) {
				r.ProposalID = nil
			},
			Exp: errors.ErrInput,
		},
		"ElectorateRef invalid": {
			Mutator: func(r *Resolution) {
				r.ElectorateRef.Version = 0
			},
			Exp: errors.ErrEmpty,
		},
	}
	for msg, spec := range specs {
		resolution := Resolution{Resolution: "123", Metadata: &weave.Metadata{Schema: 1},
			ProposalID:    weavetest.SequenceID(1),
			ElectorateRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1}}
		t.Run(msg, func(t *testing.T) {
			if spec.Mutator != nil {
				spec.Mutator(&resolution)
			}
			err := resolution.Validate()
			if !spec.Exp.Is(err) {
				t.Fatalf("check expected: %v  but got %+v", spec.Exp, err)
			}
		})
	}
}
