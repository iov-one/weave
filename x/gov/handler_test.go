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
	bobby     = aliceCond.Address()
)

func TestVote(t *testing.T) {
	proposalID := weavetest.SequenceID(1)
	specs := map[string]struct {
		Mods           func(weave.Context, *TextProposal)
		Msg            VoteMsg
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
		Exp            TallyResult
	}{
		"Vote Yes": {
			Msg: VoteMsg{ProposalId: proposalID, Selected: VoteOption_Yes, Voter: alice},
			Exp: TallyResult{TotalYes: 1},
		},
		"Vote No": {
			Msg: VoteMsg{ProposalId: proposalID, Selected: VoteOption_No, Voter: alice},
			Exp: TallyResult{TotalNo: 1},
		},
		"Vote Abstain": {
			Msg: VoteMsg{ProposalId: proposalID, Selected: VoteOption_Abstain, Voter: alice},
			Exp: TallyResult{TotalAbstain: 1},
		},
		"Vote defaults to main signer when no voter address submitted": {
			Msg: VoteMsg{ProposalId: proposalID, Selected: VoteOption_Yes},
			Exp: TallyResult{TotalYes: 1},
		},
		"Vote with invalid option": {
			Msg:            VoteMsg{ProposalId: proposalID, Selected: VoteOption_Invalid, Voter: alice},
			WantCheckErr:   errors.ErrInvalidInput,
			WantDeliverErr: errors.ErrInvalidInput,
		},
		"Unauthorized voter": {
			Msg:            VoteMsg{ProposalId: proposalID, Selected: VoteOption_Yes, Voter: weavetest.NewCondition().Address()},
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
		},
		"Vote before start date": {
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				blockTime, _ := weave.BlockTime(ctx)
				proposal.VotingStartTime = weave.AsUnixTime(blockTime.Add(time.Second))
			},
			Msg:            VoteMsg{ProposalId: proposalID, Selected: VoteOption_Yes, Voter: alice},
			WantCheckErr:   errors.ErrInvalidState,
			WantDeliverErr: errors.ErrInvalidState,
		},
		"Vote on start date": {
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				blockTime, _ := weave.BlockTime(ctx)
				proposal.VotingStartTime = weave.AsUnixTime(blockTime)
			},
			Msg:            VoteMsg{ProposalId: proposalID, Selected: VoteOption_Yes, Voter: alice},
			WantCheckErr:   errors.ErrInvalidState,
			WantDeliverErr: errors.ErrInvalidState,
		},
		"Vote on end date": {
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				blockTime, _ := weave.BlockTime(ctx)
				proposal.VotingEndTime = weave.AsUnixTime(blockTime)
			},
			Msg:            VoteMsg{ProposalId: proposalID, Selected: VoteOption_Yes, Voter: alice},
			WantCheckErr:   errors.ErrInvalidState,
			WantDeliverErr: errors.ErrInvalidState,
		},
		"Vote after end date": {
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				blockTime, _ := weave.BlockTime(ctx)
				proposal.VotingEndTime = weave.AsUnixTime(blockTime.Add(-1 * time.Second))
			},
			Msg:            VoteMsg{ProposalId: proposalID, Selected: VoteOption_Yes, Voter: alice},
			WantCheckErr:   errors.ErrInvalidState,
			WantDeliverErr: errors.ErrInvalidState,
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

			p, err := pBucket.GetTextProposal(cache, weavetest.SequenceID(1))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if exp, got := spec.Exp, p.VoteResult; exp != got {
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
			pBucket := withProposal(t, db, ctx, append([]mutator{setupForTally}, spec.Mods)...)
			cache := db.CacheWrap()

			// when check is called
			tx := &weavetest.Tx{Msg: &TallyMsg{ProposalId: weavetest.SequenceID(1)}}
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

// mutator is a call back interface to modify the passed proposal for test setup
type mutator func(weave.Context, *TextProposal)

func withProposal(t *testing.T, db store.CacheableKVStore, ctx weave.Context, mods ...mutator) *ProposalBucket {
	// setup electorate
	electorateBucket := NewElectorateBucket()
	err := electorateBucket.Save(db, electorateBucket.Build(db, &Electorate{
		Title: "fooo",
		Electors: []Elector{
			{Signature: alice, Weight: 1},
			{Signature: bobby, Weight: 10},
		}}))
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
	pBucket := NewProposalBucket()
	proposal := &TextProposal{
		Title:           "My proposal",
		ElectionRuleId:  weavetest.SequenceID(1),
		ElectorateId:    weavetest.SequenceID(1),
		VotingStartTime: weave.AsUnixTime(time.Now().Add(-1 * time.Second)),
		VotingEndTime:   weave.AsUnixTime(time.Now().Add(time.Minute)),
		Status:          TextProposal_Undefined,
	}
	for _, mod := range mods {
		if mod != nil {
			mod(ctx, proposal)
		}
	}
	err = pBucket.Save(db, pBucket.Build(db, proposal))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	return pBucket
}
