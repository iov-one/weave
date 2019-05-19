package gov

import (
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

// ctxAwareMutator is a call back interface to modify the passed proposal for test setup
type ctxAwareMutator func(weave.Context, *Proposal)

func withTextProposal(t *testing.T, db store.KVStore, ctx weave.Context, mods ...ctxAwareMutator) *ProposalBucket {
	t.Helper()
	// setup electorate
	withElectorate(t, db)
	// setup election rules
	withElectionRule(t, db)
	textOptions, _, _ := generateOptions(t)

	// adapter to call fixture mutator with context
	ctxMods := make([]func(*Proposal), len(mods)+1)
	// give a valid option here
	// TODO: merge into proposalFixture
	ctxMods[0] = func(p *Proposal) {
		p.RawOption = textOptions
	}
	for i := 0; i < len(mods); i++ {
		j := i
		ctxMods[j+1] = func(p *Proposal) {
			if mods[j] == nil {
				return
			}
			mods[j](ctx, p)
		}
	}
	pBucket := NewProposalBucket()
	proposal := proposalFixture(hAlice, ctxMods...)

	if _, err := pBucket.Create(db, &proposal); err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	return pBucket
}

func withElectorate(t *testing.T, db store.KVStore) *Electorate {
	t.Helper()
	electorate := &Electorate{
		Metadata: &weave.Metadata{Schema: 1},
		Title:    "fooo",
		Admin:    hBobby,
		Electors: []Elector{
			{Address: hAlice, Weight: 1},
			{Address: hBobby, Weight: 10},
		},
		TotalElectorateWeight: 11,
	}
	sortByAddress(electorate.Electors)
	electorateBucket := NewElectorateBucket()

	if _, err := electorateBucket.Create(db, electorate); err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	return electorate
}

func withElectionRule(t *testing.T, db store.KVStore) *ElectionRule {
	t.Helper()
	rulesBucket := NewElectionRulesBucket()
	rule := &ElectionRule{
		Metadata:          &weave.Metadata{Schema: 1},
		Title:             "barr",
		Admin:             hBobby,
		VotingPeriodHours: 1,
		Threshold:         Fraction{1, 2},
		ElectorateID:      weavetest.SequenceID(1),
	}

	if _, err := rulesBucket.Create(db, rule); err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	return rule
}

func proposalFixture(alice weave.Address, mods ...func(*Proposal)) Proposal {
	now := weave.AsUnixTime(time.Now())
	proposal := Proposal{
		Metadata: &weave.Metadata{Schema: 1},
		Common: &ProposalCommon{
			Title:           "My proposal",
			Description:     "My description",
			ElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
			ElectorateRef:   orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
			VotingStartTime: now.Add(-1 * time.Minute),
			VotingEndTime:   now.Add(time.Minute),
			SubmissionTime:  now.Add(-1 * time.Hour),
			Status:          ProposalCommon_Submitted,
			Result:          ProposalCommon_Undefined,
			Author:          alice,
			VoteState:       NewTallyResult(nil, Fraction{1, 2}, 11),
		},
		RawOption: []byte("some awesome msg to execute"),
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

// returns TextResolutionMsg, UpdateElectorateMsg, UpdateElectionRuleMsg
func generateOptions(t *testing.T) ([]byte, []byte, []byte) {
	t.Helper()

	textOpts := &ProposalOptions{
		Option: &ProposalOptions_Text{
			Text: &TextResolutionMsg{
				Metadata:   &weave.Metadata{Schema: 1},
				Resolution: "CI must be green before merging",
			},
		},
	}
	textOption, err := textOpts.Marshal()
	assert.Nil(t, err)

	electorateOpts := &ProposalOptions{
		Option: &ProposalOptions_Electorate{
			Electorate: &UpdateElectorateMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: weavetest.SequenceID(1),
				DiffElectors: []Elector{{
					Address: hCharlie,
					Weight:  22,
				}},
			},
		},
	}
	electorateOption, err := electorateOpts.Marshal()
	assert.Nil(t, err)

	ruleOpts := &ProposalOptions{
		Option: &ProposalOptions_Rule{
			Rule: &UpdateElectionRuleMsg{
				Metadata:          &weave.Metadata{Schema: 1},
				ElectionRuleID:    weavetest.SequenceID(1),
				VotingPeriodHours: 5,
				Threshold: Fraction{
					Numerator:   5,
					Denominator: 8,
				},
			},
		},
	}
	ruleOption, err := ruleOpts.Marshal()
	assert.Nil(t, err)

	return textOption, electorateOption, ruleOption
}

// returns decodable struct that fails Validate(), bytes that cannot decode
func generateInvalidOptions(t *testing.T) ([]byte, []byte) {
	t.Helper()

	missingOpts := &ProposalOptions{
		Option: &ProposalOptions_Text{
			Text: &TextResolutionMsg{
				Metadata: &weave.Metadata{Schema: 1},
			},
		},
	}
	missingOption, err := missingOpts.Marshal()
	assert.Nil(t, err)

	return missingOption, []byte("foobar")
}
