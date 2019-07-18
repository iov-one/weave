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

	// adapter to call fixture mutator with context
	ctxMods := make([]func(*Proposal), len(mods))
	for i := 0; i < len(mods); i++ {
		j := i
		ctxMods[j] = func(p *Proposal) {
			if mods[j] == nil {
				return
			}
			mods[j](ctx, p)
		}
	}
	pBucket := NewProposalBucket()
	proposal := proposalFixture(t, hAlice, ctxMods...)

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

// createElectorate persists a new Electorate in given store. Electorate is
// build using given set of addresses. First address is used as an admin
// account.
func createElectorate(t testing.TB, db store.KVStore, electors []weave.Address) {
	t.Helper()

	if len(electors) == 0 {
		t.Fatal("no electors given")
	}

	members := make([]Elector, 0)
	var total uint64
	for i, addr := range electors {
		weight := uint32(i) + 1
		members = append(members, Elector{Address: addr, Weight: weight})
		total += uint64(weight)
	}

	e := &Electorate{
		Metadata:              &weave.Metadata{Schema: 1},
		Title:                 "my electorate",
		Admin:                 electors[0],
		Electors:              members,
		TotalElectorateWeight: total,
	}
	if _, err := NewElectorateBucket().Create(db, e); err != nil {
		t.Fatalf("cannot store electorate: %+v", err)
	}
}

func withElectionRule(t *testing.T, db store.KVStore) *ElectionRule {
	t.Helper()
	rulesBucket := NewElectionRulesBucket()
	id, err := rulesBucket.NextID(db)
	assert.Nil(t, err)
	rule := &ElectionRule{
		Metadata:     &weave.Metadata{Schema: 1},
		Title:        "barr",
		Admin:        hBobby,
		VotingPeriod: weave.AsUnixDuration(time.Hour),
		Threshold:    Fraction{1, 2},
		ElectorateID: weavetest.SequenceID(1),
		Address:      Condition(id).Address(),
	}

	if _, err := rulesBucket.CreateWithID(db, id, rule); err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	return rule
}

const fixtureResolution = "Lower tx fees for all!"

func proposalFixture(t testing.TB, alice weave.Address, mods ...func(*Proposal)) Proposal {
	now := weave.AsUnixTime(time.Now())
	textOpts := &ProposalOptions{
		Option: &ProposalOptions_Text{
			Text: &CreateTextResolutionMsg{
				Metadata:   &weave.Metadata{Schema: 1},
				Resolution: fixtureResolution,
			},
		},
	}
	textOption, err := textOpts.Marshal()
	assert.Nil(t, err)

	proposal := Proposal{
		Metadata:        &weave.Metadata{Schema: 1},
		Title:           "My proposal",
		Description:     "My description",
		ElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
		ElectorateRef:   orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
		VotingStartTime: now.Add(-1 * time.Minute),
		VotingEndTime:   now.Add(time.Minute),
		SubmissionTime:  now.Add(-1 * time.Hour),
		Status:          Proposal_Submitted,
		Result:          Proposal_Undefined,
		ExecutorResult:  Proposal_NotRun,
		Author:          alice,
		VoteState:       NewTallyResult(nil, Fraction{1, 2}, 11),
		RawOption:       textOption,
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

// returns CreateTextResolutionMsg
func genTextOptions(t *testing.T) []byte {
	t.Helper()
	textOpts := &ProposalOptions{
		Option: &ProposalOptions_Text{
			Text: &CreateTextResolutionMsg{
				Metadata:   &weave.Metadata{Schema: 1},
				Resolution: "CI must be green before merging",
			},
		},
	}
	textOption, err := textOpts.Marshal()
	assert.Nil(t, err)
	return textOption
}

// returns UpdateElectorateMsg
func genElectorateOptions(t *testing.T, diff ...Elector) []byte {
	t.Helper()
	if len(diff) == 0 {
		diff = []Elector{{
			Address: hCharlie,
			Weight:  22,
		}}
	}

	electorateOpts := &ProposalOptions{
		Option: &ProposalOptions_Electorate{
			Electorate: &UpdateElectorateMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: weavetest.SequenceID(1),
				DiffElectors: diff,
			},
		},
	}
	electorateOption, err := electorateOpts.Marshal()
	assert.Nil(t, err)
	return electorateOption
}

// returns UpdateElectionRuleMsg
func genRuleOptions(t *testing.T) []byte {
	t.Helper()
	ruleOpts := &ProposalOptions{
		Option: &ProposalOptions_Rule{
			Rule: &UpdateElectionRuleMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				ElectionRuleID: weavetest.SequenceID(1),
				VotingPeriod:   weave.AsUnixDuration(5 * time.Hour),
				Threshold: Fraction{
					Numerator:   5,
					Denominator: 8,
				},
			},
		},
	}
	ruleOption, err := ruleOpts.Marshal()
	assert.Nil(t, err)
	return ruleOption
}

// returns decodable struct that fails Validate(), bytes that cannot decode
func generateInvalidOptions(t *testing.T) ([]byte, []byte) {
	t.Helper()

	missingOpts := &ProposalOptions{
		Option: &ProposalOptions_Text{
			Text: &CreateTextResolutionMsg{
				Metadata: &weave.Metadata{Schema: 1},
			},
		},
	}
	missingOption, err := missingOpts.Marshal()
	assert.Nil(t, err)

	return missingOption, []byte("foobar")
}
