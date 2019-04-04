package gov

import (
	"context"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

var (
	aliceCond = weavetest.NewCondition()
	alice     = aliceCond.Address()
	bobby     = aliceCond.Address()
)

func TestVote(t *testing.T) {
	proposalID := weavetest.SequenceID(1)
	specs := map[string]struct {
		Mods           func(proposal *TextProposal)
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
		"Vote default to main signer when voter empty": {
			Msg: VoteMsg{ProposalId: proposalID, Selected: VoteOption_Yes},
			Exp: TallyResult{TotalYes: 1},
		},
		"Vote Invalid option": {
			Msg:          VoteMsg{ProposalId: proposalID, Selected: VoteOption_Invalid, Voter: alice},
			WantCheckErr: errors.ErrInvalidInput,
		},
		"Unauthorized voter": {
			Msg:          VoteMsg{ProposalId: proposalID, Selected: VoteOption_Yes, Voter: weavetest.NewCondition().Address()},
			WantCheckErr: errors.ErrUnauthorized,
		},
		"Vote before start date": {
			Mods: func(proposal *TextProposal) {
				proposal.VotingStartTime = uint64(time.Now().Add(time.Hour).Unix())
			},
			Msg:          VoteMsg{ProposalId: proposalID, Selected: VoteOption_Yes, Voter: alice},
			WantCheckErr: errors.ErrInvalidState,
		},
		"Vote after end date": {
			Mods: func(proposal *TextProposal) {
				proposal.VotingEndTime = uint64(time.Now().Add(-1 * time.Second).Unix())
			},
			Msg:          VoteMsg{ProposalId: proposalID, Selected: VoteOption_Yes, Voter: alice},
			WantCheckErr: errors.ErrInvalidState,
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
			pBucket := WithProposal(t, db, spec.Mods)
			cache := db.CacheWrap()

			ctx := weave.WithBlockTime(context.Background(), time.Now())
			tx := &weavetest.Tx{Msg: &spec.Msg}
			if _, err := rt.Check(ctx, cache, tx); !spec.WantCheckErr.Is(err) {
				t.Fatalf("check expected: %+v  but got %+v", spec.WantCheckErr, err)
			}

			cache.Discard()
			if spec.WantCheckErr != nil {
				// Failed checks are causing the message to be ignored.
				return
			}
			if _, err := rt.Deliver(ctx, db, tx); !spec.WantDeliverErr.Is(err) {
				t.Fatalf("deliver expected: %+v  but got %+v", spec.WantCheckErr, err)
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

func WithProposal(t *testing.T, db store.CacheableKVStore, mods func(*TextProposal)) *ProposalBucket {
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
		VotingStartTime: uint64(time.Now().Add(-1 * time.Second).Unix()),
		VotingEndTime:   uint64(time.Now().Add(time.Minute).Unix()),
	}
	if mods != nil {
		mods(proposal)
	}
	err = pBucket.Save(db, pBucket.Build(db, proposal))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	return pBucket
}
