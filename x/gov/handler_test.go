package gov

import (
	"context"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/tendermint/tendermint/libs/common"
)

var (
	aliceCond = weavetest.NewCondition()
	alice     = aliceCond.Address()
	bobbyCond = weavetest.NewCondition()
	bobby     = bobbyCond.Address()
)

func TestCreateProposal(t *testing.T) {
	now := weave.AsUnixTime(time.Now())
	specs := map[string]struct {
		Mods           func(weave.Context, *TextProposal)
		Msg            CreateTextProposalMsg
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
		Exp            TextProposal
		ExpProposer    weave.Address
	}{
		"Happy path": {
			Msg: CreateTextProposalMsg{
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now.Add(time.Hour),
				ElectorateID:   weavetest.SequenceID(1),
				ElectionRuleID: weavetest.SequenceID(1),
				Author:         bobby,
			},
			Exp: TextProposal{
				Title:           "my proposal",
				Description:     "my description",
				ElectionRuleID:  weavetest.SequenceID(1),
				ElectorateID:    weavetest.SequenceID(1),
				VotingStartTime: now.Add(time.Hour),
				VotingEndTime:   now.Add(2 * time.Hour),
				Status:          TextProposal_Undefined,
				SubmissionTime:  now,
				Author:          bobby,
				VoteResult: TallyResult{
					Threshold:             Fraction{Numerator: 1, Denominator: 2},
					TotalWeightElectorate: 11,
				},
			},
			ExpProposer: bobby,
		},
		"All good with main signer as author": {
			Msg: CreateTextProposalMsg{
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now.Add(time.Hour),
				ElectorateID:   weavetest.SequenceID(1),
				ElectionRuleID: weavetest.SequenceID(1),
			},
			Exp: TextProposal{
				Title:           "my proposal",
				Description:     "my description",
				ElectionRuleID:  weavetest.SequenceID(1),
				ElectorateID:    weavetest.SequenceID(1),
				VotingStartTime: now.Add(time.Hour),
				VotingEndTime:   now.Add(2 * time.Hour),
				Status:          TextProposal_Undefined,
				SubmissionTime:  now,
				Author:          alice,
				VoteResult: TallyResult{
					Threshold:             Fraction{Numerator: 1, Denominator: 2},
					TotalWeightElectorate: 11,
				},
			},
			ExpProposer: alice,
		},
		"ElectionRuleID missing": {
			Msg: CreateTextProposalMsg{
				Title:        "my proposal",
				Description:  "my description",
				StartTime:    now.Add(time.Hour),
				ElectorateID: weavetest.SequenceID(1),
			},
			WantCheckErr:   errors.ErrInvalidInput,
			WantDeliverErr: errors.ErrInvalidInput,
		},
		"ElectionRuleID invalid": {
			Msg: CreateTextProposalMsg{
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now.Add(time.Hour),
				ElectorateID:   weavetest.SequenceID(1),
				ElectionRuleID: weavetest.SequenceID(10000),
			},
			WantCheckErr:   errors.ErrNotFound,
			WantDeliverErr: errors.ErrNotFound,
		},
		"ElectorateID missing": {
			Msg: CreateTextProposalMsg{
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now.Add(time.Hour),
				ElectionRuleID: weavetest.SequenceID(1),
			},
			WantCheckErr:   errors.ErrInvalidInput,
			WantDeliverErr: errors.ErrInvalidInput,
		},
		"ElectorateID invalid": {
			Msg: CreateTextProposalMsg{
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now.Add(time.Hour),
				ElectorateID:   weavetest.SequenceID(10000),
				ElectionRuleID: weavetest.SequenceID(1),
			},
			WantCheckErr:   errors.ErrNotFound,
			WantDeliverErr: errors.ErrNotFound,
		},
		"Author has not signed so message should be rejected": {
			Msg: CreateTextProposalMsg{
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now.Add(time.Hour),
				ElectorateID:   weavetest.SequenceID(1),
				ElectionRuleID: weavetest.SequenceID(1),
				Author:         weavetest.NewCondition().Address(),
			},
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
		},
		"Start time not in the future": {
			Msg: CreateTextProposalMsg{
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now,
				ElectorateID:   weavetest.SequenceID(1),
				ElectionRuleID: weavetest.SequenceID(1),
			},
			WantCheckErr:   errors.ErrInvalidInput,
			WantDeliverErr: errors.ErrInvalidInput,
		},
	}
	auth := &weavetest.Auth{
		Signers: []weave.Condition{aliceCond, bobbyCond},
	}
	rt := app.NewRouter()
	RegisterRoutes(rt, auth)

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			db := store.MemStore()
			// given
			ctx := weave.WithBlockTime(context.Background(), now.Time())
			pBucket := withProposal(t, db, ctx, spec.Mods)
			cache := db.CacheWrap()

			// when check is called
			tx := &weavetest.Tx{Msg: &spec.Msg}
			if _, err := rt.Check(ctx, cache, tx); !spec.WantCheckErr.Is(err) {
				t.Fatalf("check expected: %+v  but got %+v", spec.WantCheckErr, err)
			}

			cache.Discard()

			// and when deliver is called
			res, err := rt.Deliver(ctx, db, tx)
			if !spec.WantDeliverErr.Is(err) {
				t.Fatalf("deliver expected: %+v  but got %+v", spec.WantCheckErr, err)
			}
			if spec.WantDeliverErr != nil {
				return // skip further checks on expected error
			}
			// and check tags
			exp := []common.KVPair{
				{Key: []byte("proposal-id"), Value: weavetest.SequenceID(2)},
				{Key: []byte("proposer"), Value: spec.ExpProposer},
				{Key: []byte("action"), Value: []byte("create")},
			}
			if got := res.Tags; !reflect.DeepEqual(exp, got) {
				t.Errorf("expected tags %v but got %v", exp, got)
			}
			// and check persisted status
			p, err := pBucket.GetTextProposal(cache, res.Data)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if exp, got := p, &spec.Exp; !reflect.DeepEqual(exp, got) {
				t.Errorf("expected %#v but got %#v", exp, got)
			}

			cache.Discard()
		})
	}

}
func TestVote(t *testing.T) {
	proposalID := weavetest.SequenceID(1)
	specs := map[string]struct {
		Init           func(ctx weave.Context, db store.KVStore) // executed before test fixtures
		Mods           func(weave.Context, *TextProposal)        // modifies test fixtures before storing
		Msg            VoteMsg
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
		Exp            TallyResult
		ExpVotedBy     weave.Address
	}{
		"Vote Yes": {
			Msg:        VoteMsg{ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			Exp:        TallyResult{TotalYes: 1},
			ExpVotedBy: alice,
		},
		"Vote No": {
			Msg:        VoteMsg{ProposalID: proposalID, Selected: VoteOption_No, Voter: alice},
			Exp:        TallyResult{TotalNo: 1},
			ExpVotedBy: alice,
		},
		"Vote Abstain": {
			Msg:        VoteMsg{ProposalID: proposalID, Selected: VoteOption_Abstain, Voter: alice},
			Exp:        TallyResult{TotalAbstain: 1},
			ExpVotedBy: alice,
		},
		"Vote counts weights": {
			Msg:        VoteMsg{ProposalID: proposalID, Selected: VoteOption_Abstain, Voter: bobby},
			Exp:        TallyResult{TotalAbstain: 10},
			ExpVotedBy: bobby,
		},
		"Vote defaults to main signer when no voter address submitted": {
			Msg:        VoteMsg{ProposalID: proposalID, Selected: VoteOption_Yes},
			Exp:        TallyResult{TotalYes: 1},
			ExpVotedBy: alice, // main signer
		},
		"Can change vote": {
			Init: func(ctx weave.Context, db store.KVStore) {
				vBucket := NewVoteBucket()
				obj := vBucket.Build(db, proposalID, Vote{Voted: VoteOption_Yes, Elector: Elector{Signature: bobby, Weight: 10}})
				vBucket.Save(db, obj)
			},
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				proposal.VoteResult.TotalYes = 10
			},
			Msg:        VoteMsg{ProposalID: proposalID, Selected: VoteOption_No, Voter: bobby},
			Exp:        TallyResult{TotalNo: 10, TotalYes: 0},
			ExpVotedBy: bobby,
		},
		"Can resubmit vote": {
			Init: func(ctx weave.Context, db store.KVStore) {
				vBucket := NewVoteBucket()
				obj := vBucket.Build(db, proposalID, Vote{Voted: VoteOption_Yes, Elector: Elector{Signature: alice, Weight: 1}})
				vBucket.Save(db, obj)
			},
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				proposal.VoteResult.TotalYes = 1
			},
			Msg:        VoteMsg{ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			Exp:        TallyResult{TotalYes: 1},
			ExpVotedBy: alice,
		},
		"Vote with invalid option": {
			Msg:            VoteMsg{ProposalID: proposalID, Selected: VoteOption_Invalid, Voter: alice},
			WantCheckErr:   errors.ErrInvalidInput,
			WantDeliverErr: errors.ErrInvalidInput,
		},
		"Unauthorized voter": {
			Msg:            VoteMsg{ProposalID: proposalID, Selected: VoteOption_Yes, Voter: weavetest.NewCondition().Address()},
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
		},
		"Vote before start date": {
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				blockTime, _ := weave.BlockTime(ctx)
				proposal.VotingStartTime = weave.AsUnixTime(blockTime.Add(time.Second))
			},
			Msg:            VoteMsg{ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			WantCheckErr:   errors.ErrInvalidState,
			WantDeliverErr: errors.ErrInvalidState,
		},
		"Vote on start date": {
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				blockTime, _ := weave.BlockTime(ctx)
				proposal.VotingStartTime = weave.AsUnixTime(blockTime)
			},
			Msg:            VoteMsg{ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			WantCheckErr:   errors.ErrInvalidState,
			WantDeliverErr: errors.ErrInvalidState,
		},
		"Vote on end date": {
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				blockTime, _ := weave.BlockTime(ctx)
				proposal.VotingEndTime = weave.AsUnixTime(blockTime)
			},
			Msg:            VoteMsg{ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			WantCheckErr:   errors.ErrInvalidState,
			WantDeliverErr: errors.ErrInvalidState,
		},
		"Vote after end date": {
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				blockTime, _ := weave.BlockTime(ctx)
				proposal.VotingEndTime = weave.AsUnixTime(blockTime.Add(-1 * time.Second))
			},
			Msg:            VoteMsg{ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			WantCheckErr:   errors.ErrInvalidState,
			WantDeliverErr: errors.ErrInvalidState,
		},
		"Sanity check on count vote": {
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				proposal.VoteResult.TotalYes = math.MaxUint32 // not a valid setup
			},
			Msg:            VoteMsg{ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			WantDeliverErr: errors.ErrHuman,
		},
		"Sanity check on undo count vote": {
			Init: func(ctx weave.Context, db store.KVStore) {
				vBucket := NewVoteBucket()
				obj := vBucket.Build(db, proposalID, Vote{Voted: VoteOption_Yes, Elector: Elector{Signature: bobby, Weight: 10}})
				vBucket.Save(db, obj)
			},
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				proposal.VoteResult.TotalYes = 0 // not a valid setup
			},
			Msg:            VoteMsg{ProposalID: proposalID, Selected: VoteOption_No, Voter: bobby},
			WantDeliverErr: errors.ErrHuman,
		},
	}
	auth := &weavetest.Auth{
		Signer: aliceCond,
	}
	rt := app.NewRouter()
	RegisterRoutes(rt, auth)

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			db := store.MemStore()
			// given
			ctx := weave.WithBlockTime(context.Background(), time.Now().Round(time.Second))
			if spec.Init != nil {
				spec.Init(ctx, db)
			}
			pBucket := withProposal(t, db, ctx, spec.Mods)
			cache := db.CacheWrap()

			// when check
			tx := &weavetest.Tx{Msg: &spec.Msg}
			if _, err := rt.Check(ctx, cache, tx); !spec.WantCheckErr.Is(err) {
				t.Fatalf("check expected: %+v  but got %+v", spec.WantCheckErr, err)
			}

			cache.Discard()
			// and when deliver
			if _, err := rt.Deliver(ctx, db, tx); !spec.WantDeliverErr.Is(err) {
				t.Fatalf("deliver expected: %+v  but got %+v", spec.WantCheckErr, err)
			}

			if spec.WantDeliverErr != nil {
				return // skip further checks on expected error
			}
			// then tally updated
			p, err := pBucket.GetTextProposal(cache, weavetest.SequenceID(1))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if exp, got := spec.Exp, p.VoteResult; exp != got {
				t.Errorf("expected %v but got %v", exp, got)
			}
			// and vote persisted
			v, err := NewVoteBucket().GetVote(cache, proposalID, spec.ExpVotedBy)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if exp, got := spec.Msg.Selected, v.Voted; exp != got {
				t.Errorf("expected %v but got %v", exp, got)
			}
			cache.Discard()
		})
	}
}

func TestTally(t *testing.T) {
	specs := map[string]struct {
		Mods           func(weave.Context, *TextProposal)
		Src            TallyResult
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
		Exp            TextProposal_Status
	}{
		"Accepted with majority": {
			Src: TallyResult{
				TotalYes:              1,
				TotalWeightElectorate: 1,
				Threshold:             Fraction{Numerator: 1, Denominator: 2},
			},
			Exp: TextProposal_Accepted,
		},
		"Rejected with majority": {
			Src: TallyResult{
				TotalNo:               1,
				TotalWeightElectorate: 1,
				Threshold:             Fraction{Numerator: 1, Denominator: 2},
			},
			Exp: TextProposal_Rejected,
		},
		"Rejected by abstained": {
			Src: TallyResult{
				TotalAbstain:          1,
				TotalWeightElectorate: 1,
				Threshold:             Fraction{Numerator: 1, Denominator: 2},
			},
			Exp: TextProposal_Rejected,
		},
		"Rejected without voters": {
			Src: TallyResult{
				TotalWeightElectorate: 1,
				Threshold:             Fraction{Numerator: 1, Denominator: 2},
			},
			Exp: TextProposal_Rejected,
		},
		"Rejected on threshold": {
			Src: TallyResult{
				TotalYes:              1,
				TotalWeightElectorate: 2,
				Threshold:             Fraction{Numerator: 1, Denominator: 2},
			},
			Exp: TextProposal_Rejected,
		},
		"Works with high values": {
			Src: TallyResult{
				TotalYes:              math.MaxUint32,
				TotalWeightElectorate: math.MaxUint32,
				Threshold:             Fraction{Numerator: math.MaxUint32 - 1, Denominator: math.MaxUint32},
			},
			Exp: TextProposal_Accepted,
		},
		"Fails on second tally": {
			Mods: func(_ weave.Context, p *TextProposal) {
				p.Status = TextProposal_Accepted
			},
			WantCheckErr:   errors.ErrInvalidState,
			WantDeliverErr: errors.ErrInvalidState,
			Exp:            TextProposal_Accepted,
		},
		"Fails on tally before end date": {
			Mods: func(ctx weave.Context, p *TextProposal) {
				blockTime, _ := weave.BlockTime(ctx)
				p.VotingEndTime = weave.AsUnixTime(blockTime.Add(time.Second))
			},
			WantCheckErr:   errors.ErrInvalidState,
			WantDeliverErr: errors.ErrInvalidState,
			Exp:            TextProposal_Undefined,
		},
		"Fails on tally at end date": {
			Mods: func(ctx weave.Context, p *TextProposal) {
				blockTime, _ := weave.BlockTime(ctx)
				p.VotingEndTime = weave.AsUnixTime(blockTime)
			},
			WantCheckErr:   errors.ErrInvalidState,
			WantDeliverErr: errors.ErrInvalidState,
			Exp:            TextProposal_Undefined,
		},
	}
	auth := &weavetest.Auth{
		Signer: aliceCond,
	}
	rt := app.NewRouter()
	RegisterRoutes(rt, auth)

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			db := store.MemStore()
			// given
			ctx := weave.WithBlockTime(context.Background(), time.Now().Round(time.Second))
			setupForTally := func(_ weave.Context, p *TextProposal) {
				p.VoteResult = spec.Src
				blockTime, _ := weave.BlockTime(ctx)
				p.VotingEndTime = weave.AsUnixTime(blockTime.Add(-1 * time.Second))
			}
			pBucket := withProposal(t, db, ctx, append([]ctxAwareMutator{setupForTally}, spec.Mods)...)
			cache := db.CacheWrap()

			// when check is called
			tx := &weavetest.Tx{Msg: &TallyMsg{ProposalID: weavetest.SequenceID(1)}}
			if _, err := rt.Check(ctx, cache, tx); !spec.WantCheckErr.Is(err) {
				t.Fatalf("check expected: %+v  but got %+v", spec.WantCheckErr, err)
			}

			cache.Discard()

			// and when deliver is called
			res, err := rt.Deliver(ctx, db, tx)
			if !spec.WantDeliverErr.Is(err) {
				t.Fatalf("deliver expected: %+v  but got %+v", spec.WantCheckErr, err)
			}
			if spec.WantDeliverErr != nil {
				return // skip further checks on expected error
			}
			// and check tags
			exp := []common.KVPair{
				{Key: []byte("proposal-id"), Value: weavetest.SequenceID(1)},
				{Key: []byte("action"), Value: []byte("tally")},
			}
			if got := res.Tags; !reflect.DeepEqual(exp, got) {
				t.Errorf("expected tags %v but got %v", exp, got)
			}
			// and check persisted status
			p, err := pBucket.GetTextProposal(cache, weavetest.SequenceID(1))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if exp, got := spec.Exp, p.Status; exp != got {
				t.Errorf("expected %v but got %v", exp, got)
			}

			cache.Discard()
		})
	}

}

// ctxAwareMutator is a call back interface to modify the passed proposal for test setup
type ctxAwareMutator func(weave.Context, *TextProposal)

func withProposal(t *testing.T, db store.CacheableKVStore, ctx weave.Context, mods ...ctxAwareMutator) *ProposalBucket {
	// setup electorate
	electorateBucket := NewElectorateBucket()
	err := electorateBucket.Save(db, electorateBucket.Build(db, &Electorate{
		Title: "fooo",
		Electors: []Elector{
			{Signature: alice, Weight: 1},
			{Signature: bobby, Weight: 10},
		},
		TotalWeightElectorate: 11}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	// setup election rules
	rulesBucket := NewElectionRulesBucket()
	err = rulesBucket.Save(db, rulesBucket.Build(db, &ElectionRule{
		Title:             "barr",
		VotingPeriodHours: 1,
		Threshold:         Fraction{1, 2},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	// adapter to call fixture mutator with context
	ctxMods := make([]func(*TextProposal), len(mods))
	for i := 0; i < len(mods); i++ {
		j := i
		ctxMods[j] = func(p *TextProposal) {
			if mods[j] == nil {
				return
			}
			mods[j](ctx, p)
		}
	}
	pBucket := NewProposalBucket()
	proposal := textProposalFixture(ctxMods...)
	err = pBucket.Save(db, pBucket.Build(db, &proposal))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	return pBucket
}
