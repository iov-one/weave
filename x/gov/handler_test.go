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
	"github.com/iov-one/weave/weavetest/assert"
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
		Mods           ctxAwareMutator
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
				Status:          TextProposal_Submitted,
				Result:          TextProposal_Undefined,
				SubmissionTime:  now,
				Author:          bobby,
				VoteState: TallyResult{
					AcceptanceThresholdWeight: 5,
					QuorumThresholdWeight:     5,
					TotalElectorateWeight:     11,
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
				Status:          TextProposal_Submitted,
				Result:          TextProposal_Undefined,
				SubmissionTime:  now,
				Author:          alice,
				VoteState: TallyResult{
					AcceptanceThresholdWeight: 5,
					QuorumThresholdWeight:     5,
					TotalElectorateWeight:     11,
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
		"Start time too far in the future": {
			Msg: CreateTextProposalMsg{
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now.Add(7*24*time.Hour + time.Second),
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
func TestDeleteProposal(t *testing.T) {
	proposalID := weavetest.SequenceID(1)
	nonExistentProposalID := weavetest.SequenceID(2)
	specs := map[string]struct {
		Mods            func(weave.Context, *TextProposal) // modifies test fixtures before storing
		ProposalDeleted bool
		Msg             DeleteTextProposalMsg
		SignedBy        weave.Condition
		WantCheckErr    *errors.Error
		WantDeliverErr  *errors.Error
	}{
		"Happy path": {
			Msg:             DeleteTextProposalMsg{ID: proposalID},
			SignedBy:        aliceCond,
			ProposalDeleted: true,
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				proposal.VotingStartTime = weave.AsUnixTime(time.Now().Add(1 * time.Hour))
				proposal.VotingEndTime = weave.AsUnixTime(time.Now().Add(2 * time.Hour))
			},
		},
		"Proposal does not exist": {
			Msg:            DeleteTextProposalMsg{ID: nonExistentProposalID},
			SignedBy:       aliceCond,
			WantCheckErr:   errors.ErrNotFound,
			WantDeliverErr: errors.ErrNotFound,
		},
		"Delete by non-author": {
			Msg:            DeleteTextProposalMsg{ID: proposalID},
			SignedBy:       bobbyCond,
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				proposal.VotingStartTime = weave.AsUnixTime(time.Now().Add(1 * time.Hour))
				proposal.VotingEndTime = weave.AsUnixTime(time.Now().Add(2 * time.Hour))
			},
		},
		"Voting has started": {
			Msg:      DeleteTextProposalMsg{ID: proposalID},
			SignedBy: aliceCond,
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				proposal.VotingStartTime = weave.AsUnixTime(time.Now().Add(-1 * time.Hour))
				proposal.SubmissionTime = weave.AsUnixTime(time.Now().Add(-2 * time.Hour))
			},
			WantCheckErr:   errors.ErrCannotBeModified,
			WantDeliverErr: errors.ErrCannotBeModified,
		},
	}

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			db := store.MemStore()
			auth := &weavetest.Auth{
				Signer: spec.SignedBy,
			}
			rt := app.NewRouter()
			RegisterRoutes(rt, auth)

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

			// check that proposal gets deleted as expected
			p, err := pBucket.GetTextProposal(cache, weavetest.SequenceID(1))
			assert.Nil(t, err)
			if spec.ProposalDeleted {
				assert.Equal(t, p.Status, TextProposal_Withdrawn)
			} else {
				assert.Equal(t, true, p.Status != TextProposal_Withdrawn)
			}

			cache.Discard()
		})
	}
}
func TestVote(t *testing.T) {
	proposalID := weavetest.SequenceID(1)
	nonElectorCond := weavetest.NewCondition()
	nonElector := nonElectorCond.Address()
	specs := map[string]struct {
		Init           func(ctx weave.Context, db store.KVStore) // executed before test fixtures
		Mods           func(weave.Context, *TextProposal)        // modifies test fixtures before storing
		Msg            VoteMsg
		SignedBy       weave.Condition
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
		Exp            TallyResult
		ExpVotedBy     weave.Address
	}{
		"Vote Yes": {
			Msg:        VoteMsg{ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			SignedBy:   aliceCond,
			Exp:        TallyResult{TotalYes: 1, AcceptanceThresholdWeight: 5, QuorumThresholdWeight: 5, TotalElectorateWeight: 11},
			ExpVotedBy: alice,
		},
		"Vote No": {
			Msg:        VoteMsg{ProposalID: proposalID, Selected: VoteOption_No, Voter: alice},
			SignedBy:   aliceCond,
			Exp:        TallyResult{TotalNo: 1, AcceptanceThresholdWeight: 5, QuorumThresholdWeight: 5, TotalElectorateWeight: 11},
			ExpVotedBy: alice,
		},
		"Vote Abstain": {
			Msg:        VoteMsg{ProposalID: proposalID, Selected: VoteOption_Abstain, Voter: alice},
			SignedBy:   aliceCond,
			Exp:        TallyResult{TotalAbstain: 1, AcceptanceThresholdWeight: 5, QuorumThresholdWeight: 5, TotalElectorateWeight: 11},
			ExpVotedBy: alice,
		},
		"Vote counts weights": {
			Msg:        VoteMsg{ProposalID: proposalID, Selected: VoteOption_Abstain, Voter: bobby},
			SignedBy:   bobbyCond,
			Exp:        TallyResult{TotalAbstain: 10, AcceptanceThresholdWeight: 5, QuorumThresholdWeight: 5, TotalElectorateWeight: 11},
			ExpVotedBy: bobby,
		},
		"Vote defaults to main signer when no voter address submitted": {
			Msg:        VoteMsg{ProposalID: proposalID, Selected: VoteOption_Yes},
			SignedBy:   aliceCond,
			Exp:        TallyResult{TotalYes: 1, AcceptanceThresholdWeight: 5, QuorumThresholdWeight: 5, TotalElectorateWeight: 11},
			ExpVotedBy: alice,
		},
		"Can change vote": {
			Init: func(ctx weave.Context, db store.KVStore) {
				vBucket := NewVoteBucket()
				obj := vBucket.Build(db, proposalID, Vote{Voted: VoteOption_Yes, Elector: Elector{Address: bobby, Weight: 10}})
				vBucket.Save(db, obj)
			},
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				proposal.VoteState.TotalYes = 10
			},
			Msg:        VoteMsg{ProposalID: proposalID, Selected: VoteOption_No, Voter: bobby},
			SignedBy:   bobbyCond,
			Exp:        TallyResult{TotalNo: 10, TotalYes: 0, AcceptanceThresholdWeight: 5, QuorumThresholdWeight: 5, TotalElectorateWeight: 11},
			ExpVotedBy: bobby,
		},
		"Can resubmit vote": {
			Init: func(ctx weave.Context, db store.KVStore) {
				vBucket := NewVoteBucket()
				obj := vBucket.Build(db, proposalID, Vote{Voted: VoteOption_Yes, Elector: Elector{Address: alice, Weight: 1}})
				vBucket.Save(db, obj)
			},
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				proposal.VoteState.TotalYes = 1
			},
			Msg:        VoteMsg{ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			SignedBy:   aliceCond,
			Exp:        TallyResult{TotalYes: 1, AcceptanceThresholdWeight: 5, QuorumThresholdWeight: 5, TotalElectorateWeight: 11},
			ExpVotedBy: alice,
		},
		"Voter must sign": {
			Msg:            VoteMsg{ProposalID: proposalID, Selected: VoteOption_Yes, Voter: bobby},
			SignedBy:       aliceCond,
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
		},
		"Vote with invalid option": {
			Msg:            VoteMsg{ProposalID: proposalID, Selected: VoteOption_Invalid, Voter: alice},
			SignedBy:       aliceCond,
			WantCheckErr:   errors.ErrInvalidInput,
			WantDeliverErr: errors.ErrInvalidInput,
		},
		"Voter not in electorate must be rejected": {
			Msg:            VoteMsg{ProposalID: proposalID, Selected: VoteOption_Yes, Voter: nonElector},
			SignedBy:       nonElectorCond,
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
		},
		"Vote before start date": {
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				blockTime, _ := weave.BlockTime(ctx)
				proposal.VotingStartTime = weave.AsUnixTime(blockTime.Add(time.Second))
			},
			Msg:            VoteMsg{ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			SignedBy:       aliceCond,
			WantCheckErr:   errors.ErrInvalidState,
			WantDeliverErr: errors.ErrInvalidState,
		},
		"Vote on start date": {
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				blockTime, _ := weave.BlockTime(ctx)
				proposal.VotingStartTime = weave.AsUnixTime(blockTime)
			},
			Msg:            VoteMsg{ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			SignedBy:       aliceCond,
			WantCheckErr:   errors.ErrInvalidState,
			WantDeliverErr: errors.ErrInvalidState,
		},
		"Vote on end date": {
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				blockTime, _ := weave.BlockTime(ctx)
				proposal.VotingEndTime = weave.AsUnixTime(blockTime)
			},
			Msg:            VoteMsg{ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			SignedBy:       aliceCond,
			WantCheckErr:   errors.ErrInvalidState,
			WantDeliverErr: errors.ErrInvalidState,
		},
		"Vote after end date": {
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				blockTime, _ := weave.BlockTime(ctx)
				proposal.VotingEndTime = weave.AsUnixTime(blockTime.Add(-1 * time.Second))
			},
			Msg:            VoteMsg{ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			SignedBy:       aliceCond,
			WantCheckErr:   errors.ErrInvalidState,
			WantDeliverErr: errors.ErrInvalidState,
		},
		"Vote on withdrawn proposal must fail": {
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				proposal.Status = TextProposal_Withdrawn
			},
			Msg:            VoteMsg{ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			SignedBy:       aliceCond,
			WantCheckErr:   errors.ErrInvalidState,
			WantDeliverErr: errors.ErrInvalidState,
		},
		"Vote on closed proposal must fail": {
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				proposal.Status = TextProposal_Closed
			},
			Msg:            VoteMsg{ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			SignedBy:       aliceCond,
			WantCheckErr:   errors.ErrInvalidState,
			WantDeliverErr: errors.ErrInvalidState,
		},
		"Sanity check on count vote": {
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				// not a valid setup
				proposal.VoteState.TotalYes = math.MaxUint64
				proposal.VoteState.TotalElectorateWeight = math.MaxUint64
			},
			Msg:            VoteMsg{ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			SignedBy:       aliceCond,
			WantDeliverErr: errors.ErrHuman,
		},
		"Sanity check on undo count vote": {
			Init: func(ctx weave.Context, db store.KVStore) {
				vBucket := NewVoteBucket()
				obj := vBucket.Build(db, proposalID, Vote{Voted: VoteOption_Yes, Elector: Elector{Address: bobby, Weight: 10}})
				vBucket.Save(db, obj)
			},
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				// not a valid setup
				proposal.VoteState.TotalYes = 0
				proposal.VoteState.TotalElectorateWeight = math.MaxUint64
			},
			Msg:            VoteMsg{ProposalID: proposalID, Selected: VoteOption_No, Voter: bobby},
			SignedBy:       bobbyCond,
			WantDeliverErr: errors.ErrHuman,
		},
	}

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			db := store.MemStore()
			auth := &weavetest.Auth{
				Signer: spec.SignedBy,
			}
			rt := app.NewRouter()
			RegisterRoutes(rt, auth)

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
			if exp, got := spec.Exp, p.VoteState; exp != got {
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
	type tallySetup struct {
		quorum                *Fraction
		threshold             Fraction
		totalWeightElectorate uint64
		yes, no, abstain      uint64
	}
	specs := map[string]struct {
		Mods           func(weave.Context, *TextProposal)
		Src            tallySetup
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
		ExpResult      TextProposal_Result
	}{
		"Accepted with electorate majority": {
			Src: tallySetup{
				yes:                   5,
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: TextProposal_Accepted,
		},
		"Accepted with all yes votes required": {
			Src: tallySetup{
				yes:                   9,
				threshold:             Fraction{Numerator: 1, Denominator: 1},
				totalWeightElectorate: 9,
			},
			ExpResult: TextProposal_Accepted,
		},
		"Rejected without enough votes": {
			Src: tallySetup{
				yes:                   4,
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: TextProposal_Rejected,
		},
		"Rejected when acceptance weight not reached": {
			Src: tallySetup{
				yes:                   4,
				no:                    1,
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: TextProposal_Rejected,
		},
		"Rejected on acceptance threshold": {
			Src: tallySetup{
				yes:                   4,
				no:                    1,
				abstain:               3,
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: TextProposal_Rejected,
		},
		"Rejected without voters": {
			Src: tallySetup{
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 2,
			},
			ExpResult: TextProposal_Rejected,
		},
		"Rejected without enough votes: 2/3": {
			Src: tallySetup{
				yes:                   6,
				threshold:             Fraction{Numerator: 2, Denominator: 3},
				totalWeightElectorate: 9,
			},
			ExpResult: TextProposal_Rejected,
		},
		"Accepted with quorum and acceptance thresholds exceeded: 5/9": {
			Src: tallySetup{
				yes:                   5,
				quorum:                &Fraction{Numerator: 1, Denominator: 2},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: TextProposal_Accepted,
		},
		"Rejected with quorum thresholds not exceeded": {
			Src: tallySetup{
				yes:                   4,
				quorum:                &Fraction{Numerator: 1, Denominator: 2},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: TextProposal_Rejected,
		},
		"Accepted with quorum and acceptance thresholds exceeded: 4/9": {
			Src: tallySetup{
				yes:                   4,
				no:                    1,
				quorum:                &Fraction{Numerator: 1, Denominator: 2},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: TextProposal_Accepted,
		},
		"Accepted with quorum and acceptance thresholds exceeded: 4/9 and majority No": {
			Src: tallySetup{
				yes:                   4,
				no:                    5,
				quorum:                &Fraction{Numerator: 1, Denominator: 2},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: TextProposal_Accepted,
		},
		"Accepted with quorum and acceptance thresholds exceeded: 3/9": {
			Src: tallySetup{
				yes:                   3,
				abstain:               2,
				quorum:                &Fraction{Numerator: 1, Denominator: 2},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: TextProposal_Accepted,
		},
		"Rejected with quorum but not acceptance thresholds exceeded: 2/9": {
			Src: tallySetup{
				yes:                   2,
				abstain:               3,
				quorum:                &Fraction{Numerator: 1, Denominator: 2},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: TextProposal_Rejected,
		},
		"Accepted with quorum and acceptance thresholds exceeded: 6/9": {
			Src: tallySetup{
				yes:                   6,
				abstain:               1,
				quorum:                &Fraction{Numerator: 2, Denominator: 3},
				threshold:             Fraction{Numerator: 2, Denominator: 3},
				totalWeightElectorate: 9,
			},
			ExpResult: TextProposal_Accepted,
		},
		"Accepted with quorum and acceptance thresholds exceeded: 9/9": {
			Src: tallySetup{
				yes:                   9,
				quorum:                &Fraction{Numerator: 1, Denominator: 1},
				threshold:             Fraction{Numerator: 1, Denominator: 1},
				totalWeightElectorate: 9,
			},
			ExpResult: TextProposal_Accepted,
		},
		"Works with high values": {
			Src: tallySetup{
				yes:                   math.MaxUint64,
				threshold:             Fraction{Numerator: math.MaxUint32 - 1, Denominator: math.MaxUint32},
				totalWeightElectorate: math.MaxUint64,
			},
			ExpResult: TextProposal_Accepted,
		},
		"Fails on second tally": {
			Mods: func(_ weave.Context, p *TextProposal) {
				p.Status = TextProposal_Closed
			},
			Src: tallySetup{
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 11,
			},
			WantCheckErr:   errors.ErrInvalidState,
			WantDeliverErr: errors.ErrInvalidState,
			ExpResult:      TextProposal_Accepted,
		},
		"Fails on tally before end date": {
			Mods: func(ctx weave.Context, p *TextProposal) {
				blockTime, _ := weave.BlockTime(ctx)
				p.VotingEndTime = weave.AsUnixTime(blockTime.Add(time.Second))
			},
			Src: tallySetup{
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 11,
			},
			WantCheckErr:   errors.ErrInvalidState,
			WantDeliverErr: errors.ErrInvalidState,
			ExpResult:      TextProposal_Undefined,
		},
		"Fails on tally at end date": {
			Mods: func(ctx weave.Context, p *TextProposal) {
				blockTime, _ := weave.BlockTime(ctx)
				p.VotingEndTime = weave.AsUnixTime(blockTime)
			},
			Src: tallySetup{
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 11,
			},
			WantCheckErr:   errors.ErrInvalidState,
			WantDeliverErr: errors.ErrInvalidState,
			ExpResult:      TextProposal_Undefined,
		},
		"Fails on withdrawn proposal": {
			Mods: func(ctx weave.Context, p *TextProposal) {
				p.Status = TextProposal_Withdrawn
			},
			Src: tallySetup{
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 11,
			},
			WantCheckErr:   errors.ErrInvalidState,
			WantDeliverErr: errors.ErrInvalidState,
			ExpResult:      TextProposal_Undefined,
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
				p.VoteState = NewTallyResult(spec.Src.quorum, spec.Src.threshold, spec.Src.totalWeightElectorate)
				p.VoteState.TotalYes = spec.Src.yes
				p.VoteState.TotalNo = spec.Src.no
				p.VoteState.TotalAbstain = spec.Src.abstain
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
			// and check persisted result
			p, err := pBucket.GetTextProposal(cache, weavetest.SequenceID(1))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if exp, got := spec.ExpResult, p.Result; exp != got {
				t.Errorf("expected %v but got %v: vote state: %#v", exp, got, p.VoteState)
			}
			if exp, got := TextProposal_Closed, p.Status; exp != got {
				t.Errorf("expected %v but got %v", exp, got)
			}
			cache.Discard()
		})
	}
}

func TestUpdateElectorate(t *testing.T) {
	electorateID := weavetest.SequenceID(1)

	specs := map[string]struct {
		Msg            UpdateElectorateMsg
		SignedBy       weave.Condition
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
		ExpModel       *Electorate
		WithProposal   bool            // enables the usage of mods to create a proposal
		Mods           ctxAwareMutator // modifies TextProposal test fixtures before storing
	}{
		"All good with update by owner": {
			Msg: UpdateElectorateMsg{
				ElectorateID: electorateID,
				Electors:     []Elector{{Address: alice, Weight: 22}},
			},
			SignedBy: bobbyCond,
			ExpModel: &Electorate{
				Admin:                 bobby,
				Title:                 "fooo",
				Electors:              []Elector{{Address: alice, Weight: 22}},
				TotalElectorateWeight: 22,
			},
		},
		"Update by non owner should fail": {
			Msg: UpdateElectorateMsg{
				ElectorateID: electorateID,
				Electors:     []Elector{{Address: alice, Weight: 22}},
			},
			SignedBy:       aliceCond,
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
		},
		"Update with open proposal should fail": {
			Msg: UpdateElectorateMsg{
				ElectorateID: electorateID,
				Electors:     []Elector{{Address: alice, Weight: 22}},
			},
			SignedBy: bobbyCond,
			ExpModel: &Electorate{
				Admin:                 bobby,
				Title:                 "fooo",
				Electors:              []Elector{{Address: alice, Weight: 22}},
				TotalElectorateWeight: 22,
			},
			WithProposal: true,
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				proposal.Status = TextProposal_Submitted
			},
			WantDeliverErr: errors.ErrInvalidState,
		},
		"Update with closed proposal should succeed": {
			Msg: UpdateElectorateMsg{
				ElectorateID: electorateID,
				Electors:     []Elector{{Address: alice, Weight: 22}},
			},
			SignedBy: bobbyCond,
			ExpModel: &Electorate{
				Admin:                 bobby,
				Title:                 "fooo",
				Electors:              []Elector{{Address: alice, Weight: 22}},
				TotalElectorateWeight: 22,
			},
			WithProposal: true,
			Mods: func(ctx weave.Context, proposal *TextProposal) {
				proposal.Status = TextProposal_Closed
			},
		},
		"Update with too many electors should fail": {
			Msg: UpdateElectorateMsg{
				ElectorateID: electorateID,
				Electors:     buildElectors(2001),
			},
			SignedBy:       bobbyCond,
			WantCheckErr:   errors.ErrInvalidInput,
			WantDeliverErr: errors.ErrInvalidInput,
		},
		"Update without electors should fail": {
			Msg: UpdateElectorateMsg{
				ElectorateID: electorateID,
			},
			SignedBy:       bobbyCond,
			WantCheckErr:   errors.ErrEmpty,
			WantDeliverErr: errors.ErrEmpty,
		},
		"Duplicate electors should fail": {
			Msg: UpdateElectorateMsg{
				ElectorateID: electorateID,
				Electors:     []Elector{{Address: alice, Weight: 1}, {Address: alice, Weight: 2}},
			},
			SignedBy:       bobbyCond,
			WantCheckErr:   errors.ErrInvalidInput,
			WantDeliverErr: errors.ErrInvalidInput,
		},
		"Empty address in electors should fail": {
			Msg: UpdateElectorateMsg{
				ElectorateID: electorateID,
				Electors:     []Elector{{Address: weave.Address{}, Weight: 1}},
			},
			SignedBy:       bobbyCond,
			WantCheckErr:   errors.ErrEmpty,
			WantDeliverErr: errors.ErrEmpty,
		},
	}
	bucket := NewElectorateBucket()
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			auth := &weavetest.Auth{
				Signer: spec.SignedBy,
			}
			rt := app.NewRouter()
			RegisterRoutes(rt, auth)
			db := store.MemStore()
			withElectorate(t, db)
			if spec.WithProposal {
				withProposal(t, db, nil, spec.Mods)
			}
			cache := db.CacheWrap()

			ctx := context.Background()
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
			e, err := bucket.GetElectorate(db, res.Data)
			if err != nil {
				t.Fatalf("unexpected error: %+v", err)
			}
			if exp, got := spec.ExpModel, e; !reflect.DeepEqual(exp, got) {
				t.Errorf("expected %v but got %v", exp, got)
			}
		})
	}
}

func TestUpdateElectionRules(t *testing.T) {
	electionRulesID := weavetest.SequenceID(1)

	specs := map[string]struct {
		Msg            UpdateElectionRuleMsg
		SignedBy       weave.Condition
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
		ExpModel       *ElectionRule
	}{
		"All good with update by owner": {
			Msg: UpdateElectionRuleMsg{
				ElectionRuleID:    electionRulesID,
				VotingPeriodHours: 12,
				Threshold:         Fraction{Numerator: 2, Denominator: 3},
			},
			SignedBy: bobbyCond,
			ExpModel: &ElectionRule{
				Admin:             bobby,
				Title:             "barr",
				VotingPeriodHours: 12,
				Threshold:         Fraction{Numerator: 2, Denominator: 3},
			},
		},
		"Update with max voting time": {
			Msg: UpdateElectionRuleMsg{
				ElectionRuleID:    electionRulesID,
				VotingPeriodHours: 4 * 7 * 24,
				Threshold:         Fraction{Numerator: 2, Denominator: 3},
			},
			SignedBy: bobbyCond,
			ExpModel: &ElectionRule{
				Admin:             bobby,
				Title:             "barr",
				VotingPeriodHours: 4 * 7 * 24,
				Threshold:         Fraction{Numerator: 2, Denominator: 3},
			},
		},
		"Update by non owner should fail": {
			Msg: UpdateElectionRuleMsg{
				ElectionRuleID:    electionRulesID,
				VotingPeriodHours: 12,
				Threshold:         Fraction{Numerator: 2, Denominator: 3},
			},
			SignedBy:       aliceCond,
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
		},
		"Threshold must be valid": {
			Msg: UpdateElectionRuleMsg{
				ElectionRuleID:    electionRulesID,
				VotingPeriodHours: 12,
				Threshold:         Fraction{Numerator: 3, Denominator: 2},
			},
			SignedBy:       bobbyCond,
			WantCheckErr:   errors.ErrInvalidInput,
			WantDeliverErr: errors.ErrInvalidInput,
		},
		"voting period hours must not be empty": {
			Msg: UpdateElectionRuleMsg{
				ElectionRuleID:    electionRulesID,
				VotingPeriodHours: 0,
				Threshold:         Fraction{Numerator: 1, Denominator: 2},
			},
			SignedBy:       bobbyCond,
			WantCheckErr:   errors.ErrInvalidInput,
			WantDeliverErr: errors.ErrInvalidInput,
		},
		"voting period hours must not exceed max": {
			Msg: UpdateElectionRuleMsg{
				ElectionRuleID:    electionRulesID,
				VotingPeriodHours: 4*7*24 + 1,
				Threshold:         Fraction{Numerator: 1, Denominator: 2},
			},
			SignedBy:       bobbyCond,
			WantCheckErr:   errors.ErrInvalidInput,
			WantDeliverErr: errors.ErrInvalidInput,
		},
	}
	bucket := NewElectionRulesBucket()
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			auth := &weavetest.Auth{
				Signer: spec.SignedBy,
			}
			rt := app.NewRouter()
			RegisterRoutes(rt, auth)
			db := store.MemStore()
			withElectionRule(t, db)
			cache := db.CacheWrap()

			ctx := context.Background()
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
			e, err := bucket.GetElectionRule(db, res.Data)
			if err != nil {
				t.Fatalf("unexpected error: %+v", err)
			}
			if exp, got := spec.ExpModel, e; !reflect.DeepEqual(exp, got) {
				t.Errorf("expected %v but got %v", exp, got)
			}
		})
	}
}

// ctxAwareMutator is a call back interface to modify the passed proposal for test setup
type ctxAwareMutator func(weave.Context, *TextProposal)

func withProposal(t *testing.T, db store.CacheableKVStore, ctx weave.Context, mods ...ctxAwareMutator) *ProposalBucket {
	// setup electorate
	withElectorate(t, db)
	// setup election rules
	withElectionRule(t, db)
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
	pObj, err := pBucket.Build(db, &proposal)
	assert.Nil(t, err)
	err = pBucket.Save(db, pObj)
	assert.Nil(t, err)

	return pBucket
}

func withElectorate(t *testing.T, db store.CacheableKVStore) *Electorate {
	electorate := &Electorate{
		Title: "fooo",
		Admin: bobby,
		Electors: []Elector{
			{Address: alice, Weight: 1},
			{Address: bobby, Weight: 10},
		},
		TotalElectorateWeight: 11,
	}
	electorateBucket := NewElectorateBucket()
	eObj, err := electorateBucket.Build(db, electorate)
	assert.Nil(t, err)
	err = electorateBucket.Save(db, eObj)
	assert.Nil(t, err)
	return electorate
}

func withElectionRule(t *testing.T, db store.CacheableKVStore) *ElectionRule {
	rulesBucket := NewElectionRulesBucket()
	rule := &ElectionRule{
		Title:             "barr",
		Admin:             bobby,
		VotingPeriodHours: 1,
		Threshold:         Fraction{1, 2},
	}

	rObj, err := rulesBucket.Build(db, rule)
	assert.Nil(t, err)
	err = rulesBucket.Save(db, rObj)
	assert.Nil(t, err)

	return rule
}
