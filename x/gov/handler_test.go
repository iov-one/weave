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
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

var (
	aliceCond   = weavetest.NewCondition()
	alice       = aliceCond.Address()
	bobbyCond   = weavetest.NewCondition()
	bobby       = bobbyCond.Address()
	charlieCond = weavetest.NewCondition()
	charlie     = charlieCond.Address()
)

func TestCreateTextProposal(t *testing.T) {
	now := weave.AsUnixTime(time.Now())
	specs := map[string]struct {
		Init           func(ctx weave.Context, db store.KVStore) // executed before test fixtures
		Msg            CreateTextProposalMsg
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
		Exp            Proposal
		ExpProposer    weave.Address
	}{
		"Happy path": {
			Msg: CreateTextProposalMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now.Add(time.Hour),
				ElectorateID:   weavetest.SequenceID(1),
				ElectionRuleID: weavetest.SequenceID(1),
				Author:         bobby,
			},
			Exp: Proposal{
				Metadata:        &weave.Metadata{Schema: 1},
				Type:            Proposal_Text,
				Title:           "my proposal",
				Description:     "my description",
				ElectionRuleID:  weavetest.SequenceID(1),
				ElectorateID:    weavetest.SequenceID(1),
				VotingStartTime: now.Add(time.Hour),
				VotingEndTime:   now.Add(2 * time.Hour),
				Status:          Proposal_Submitted,
				Result:          Proposal_Undefined,
				SubmissionTime:  now,
				Author:          bobby,
				VoteState: TallyResult{
					Threshold:             Fraction{Numerator: 1, Denominator: 2},
					TotalElectorateWeight: 11,
				},
				Details: &Proposal_TextDetails{&TextProposalPayload{}},
			},
			ExpProposer: bobby,
		},
		"All good with main signer as author": {
			Msg: CreateTextProposalMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now.Add(time.Hour),
				ElectorateID:   weavetest.SequenceID(1),
				ElectionRuleID: weavetest.SequenceID(1),
			},
			Exp: Proposal{
				Metadata:        &weave.Metadata{Schema: 1},
				Type:            Proposal_Text,
				Title:           "my proposal",
				Description:     "my description",
				ElectionRuleID:  weavetest.SequenceID(1),
				ElectorateID:    weavetest.SequenceID(1),
				VotingStartTime: now.Add(time.Hour),
				VotingEndTime:   now.Add(2 * time.Hour),
				Status:          Proposal_Submitted,
				Result:          Proposal_Undefined,
				SubmissionTime:  now,
				Author:          alice,
				VoteState: TallyResult{
					Threshold:             Fraction{Numerator: 1, Denominator: 2},
					TotalElectorateWeight: 11,
				},
				Details: &Proposal_TextDetails{&TextProposalPayload{}},
			},
			ExpProposer: alice,
		},
		"ElectionRuleID missing": {
			Msg: CreateTextProposalMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "my proposal",
				Description:  "my description",
				StartTime:    now.Add(time.Hour),
				ElectorateID: weavetest.SequenceID(1),
			},
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"ElectionRuleID invalid": {
			Msg: CreateTextProposalMsg{
				Metadata:       &weave.Metadata{Schema: 1},
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
				Metadata:       &weave.Metadata{Schema: 1},
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now.Add(time.Hour),
				ElectionRuleID: weavetest.SequenceID(1),
			},
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"ElectorateID invalid": {
			Msg: CreateTextProposalMsg{
				Metadata:       &weave.Metadata{Schema: 1},
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
				Metadata:       &weave.Metadata{Schema: 1},
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
				Metadata:       &weave.Metadata{Schema: 1},
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now,
				ElectorateID:   weavetest.SequenceID(1),
				ElectionRuleID: weavetest.SequenceID(1),
			},
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"Start time too far in the future": {
			Msg: CreateTextProposalMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now.Add(7*24*time.Hour + time.Second),
				ElectorateID:   weavetest.SequenceID(1),
				ElectionRuleID: weavetest.SequenceID(1),
			},
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"Update electorate proposal exists": {
			Init: func(ctx weave.Context, db store.KVStore) {
				bucket := NewProposalBucket()
				blocking := updateElectoreateProposalFixture()

				if _, err := bucket.Create(db, &blocking); err != nil {
					t.Fatalf("unexpected error: %+v", err)
				}
			},
			Msg: CreateTextProposalMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now.Add(time.Hour),
				ElectorateID:   weavetest.SequenceID(1),
				ElectionRuleID: weavetest.SequenceID(1),
				Author:         bobby,
			},
			WantDeliverErr: errors.ErrState,
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
			migration.MustInitPkg(db, packageName)

			// given
			ctx := weave.WithBlockTime(context.Background(), now.Time())
			if spec.Init != nil {
				spec.Init(ctx, db)
			}
			// setup election rules
			withElectionRule(t, db)
			// setup electorate
			withElectorate(t, db)

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
			// and check persisted status
			p, err := NewProposalBucket().GetProposal(cache, res.Data)
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

func TestCreateElectorateUpdateProposal(t *testing.T) {
	now := weave.AsUnixTime(time.Now())
	specs := map[string]struct {
		Init           func(ctx weave.Context, db store.KVStore) // executed before test fixtures
		Mods           ctxAwareMutator
		Msg            CreateElectorateUpdateProposalMsg
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
		Exp            Proposal
		ExpProposer    weave.Address
	}{
		"Happy path": {
			Msg: CreateElectorateUpdateProposalMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "my proposal",
				Description:  "my description",
				StartTime:    now.Add(time.Hour),
				ElectorateID: weavetest.SequenceID(1),
				DiffElectors: []Elector{{alice, 10}},
				Author:       bobby,
			},
			Exp: Proposal{
				Metadata:        &weave.Metadata{Schema: 1},
				Type:            Proposal_UpdateElectorate,
				Title:           "my proposal",
				Description:     "my description",
				ElectionRuleID:  weavetest.SequenceID(1),
				ElectorateID:    weavetest.SequenceID(1),
				VotingStartTime: now.Add(time.Hour),
				VotingEndTime:   now.Add(2 * time.Hour),
				Status:          Proposal_Submitted,
				Result:          Proposal_Undefined,
				SubmissionTime:  now,
				Author:          bobby,
				VoteState: TallyResult{
					Threshold:             Fraction{Numerator: 1, Denominator: 2},
					TotalElectorateWeight: 11,
				},
				Details: &Proposal_ElectorateUpdateDetails{&ElectorateUpdatePayload{
					[]Elector{{Address: alice, Weight: 10}},
				}},
			},
			ExpProposer: bobby,
		},
		"All good with main signer as author": {
			Msg: CreateElectorateUpdateProposalMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "my proposal",
				Description:  "my description",
				StartTime:    now.Add(time.Hour),
				ElectorateID: weavetest.SequenceID(1),
				DiffElectors: []Elector{{alice, 10}},
			},
			Exp: Proposal{
				Metadata:        &weave.Metadata{Schema: 1},
				Type:            Proposal_UpdateElectorate,
				Title:           "my proposal",
				Description:     "my description",
				ElectionRuleID:  weavetest.SequenceID(1),
				ElectorateID:    weavetest.SequenceID(1),
				VotingStartTime: now.Add(time.Hour),
				VotingEndTime:   now.Add(2 * time.Hour),
				Status:          Proposal_Submitted,
				Result:          Proposal_Undefined,
				SubmissionTime:  now,
				Author:          alice,
				VoteState: TallyResult{
					Threshold:             Fraction{Numerator: 1, Denominator: 2},
					TotalElectorateWeight: 11,
				},
				Details: &Proposal_ElectorateUpdateDetails{&ElectorateUpdatePayload{
					[]Elector{{Address: alice, Weight: 10}},
				}},
			},
			ExpProposer: alice,
		},
		"ElectorateID missing": {
			Msg: CreateElectorateUpdateProposalMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "my proposal",
				Description:  "my description",
				StartTime:    now.Add(time.Hour),
				DiffElectors: []Elector{{alice, 10}},
			},
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"ElectorateID invalid": {
			Msg: CreateElectorateUpdateProposalMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "my proposal",
				Description:  "my description",
				StartTime:    now.Add(time.Hour),
				ElectorateID: weavetest.SequenceID(10000),
				DiffElectors: []Elector{{alice, 10}},
			},
			WantCheckErr:   errors.ErrNotFound,
			WantDeliverErr: errors.ErrNotFound,
		},
		"Author has not signed so message should be rejected": {
			Msg: CreateElectorateUpdateProposalMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "my proposal",
				Description:  "my description",
				StartTime:    now.Add(time.Hour),
				ElectorateID: weavetest.SequenceID(1),
				DiffElectors: []Elector{{alice, 10}},
				Author:       weavetest.NewCondition().Address(),
			},
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
		},
		"Start time not in the future": {
			Msg: CreateElectorateUpdateProposalMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "my proposal",
				Description:  "my description",
				StartTime:    now,
				ElectorateID: weavetest.SequenceID(1),
				DiffElectors: []Elector{{alice, 10}},
			},
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"Start time too far in the future": {
			Msg: CreateElectorateUpdateProposalMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "my proposal",
				Description:  "my description",
				StartTime:    now.Add(7*24*time.Hour + time.Second),
				ElectorateID: weavetest.SequenceID(1),
				DiffElectors: []Elector{{alice, 10}},
			},
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"open text proposal exists": {
			Init: func(ctx weave.Context, db store.KVStore) {
				bucket := NewProposalBucket()
				blocking := textProposalFixture()
				if _, err := bucket.Create(db, &blocking); err != nil {
					t.Fatalf("unexpected error: %+v", err)
				}
			},
			Msg: CreateElectorateUpdateProposalMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				Title:        "my proposal",
				Description:  "my description",
				StartTime:    now.Add(time.Hour),
				ElectorateID: weavetest.SequenceID(1),
				DiffElectors: []Elector{{alice, 10}},
				Author:       bobby,
			},
			WantDeliverErr: errors.ErrState,
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
			migration.MustInitPkg(db, packageName)

			// given
			ctx := weave.WithBlockTime(context.Background(), now.Time())
			if spec.Init != nil {
				spec.Init(ctx, db)
			}
			// setup election rules
			withElectionRule(t, db)
			// setup electorate
			withElectorate(t, db)

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
			// and check persisted status
			p, err := NewProposalBucket().GetProposal(cache, res.Data)
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
		Mods            func(weave.Context, *Proposal) // modifies test fixtures before storing
		ProposalDeleted bool
		Msg             DeleteProposalMsg
		SignedBy        weave.Condition
		WantCheckErr    *errors.Error
		WantDeliverErr  *errors.Error
	}{
		"Happy path": {
			Msg:             DeleteProposalMsg{Metadata: &weave.Metadata{Schema: 1}, ID: proposalID},
			SignedBy:        aliceCond,
			ProposalDeleted: true,
			Mods: func(ctx weave.Context, proposal *Proposal) {
				proposal.VotingStartTime = weave.AsUnixTime(time.Now().Add(1 * time.Hour))
				proposal.VotingEndTime = weave.AsUnixTime(time.Now().Add(2 * time.Hour))
			},
		},
		"Proposal does not exist": {
			Msg:            DeleteProposalMsg{Metadata: &weave.Metadata{Schema: 1}, ID: nonExistentProposalID},
			SignedBy:       aliceCond,
			WantCheckErr:   errors.ErrNotFound,
			WantDeliverErr: errors.ErrNotFound,
		},
		"Delete by non-author": {
			Msg:            DeleteProposalMsg{Metadata: &weave.Metadata{Schema: 1}, ID: proposalID},
			SignedBy:       bobbyCond,
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
			Mods: func(ctx weave.Context, proposal *Proposal) {
				proposal.VotingStartTime = weave.AsUnixTime(time.Now().Add(1 * time.Hour))
				proposal.VotingEndTime = weave.AsUnixTime(time.Now().Add(2 * time.Hour))
			},
		},
		"Voting has started": {
			Msg:      DeleteProposalMsg{Metadata: &weave.Metadata{Schema: 1}, ID: proposalID},
			SignedBy: aliceCond,
			Mods: func(ctx weave.Context, proposal *Proposal) {
				proposal.VotingStartTime = weave.AsUnixTime(time.Now().Add(-1 * time.Hour))
				proposal.SubmissionTime = weave.AsUnixTime(time.Now().Add(-2 * time.Hour))
			},
			WantCheckErr:   errors.ErrImmutable,
			WantDeliverErr: errors.ErrImmutable,
		},
	}

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, packageName)

			auth := &weavetest.Auth{
				Signer: spec.SignedBy,
			}
			rt := app.NewRouter()
			RegisterRoutes(rt, auth)

			// given
			ctx := weave.WithBlockTime(context.Background(), time.Now().Round(time.Second))
			pBucket := withTextProposal(t, db, ctx, spec.Mods)
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
			p, err := pBucket.GetProposal(cache, weavetest.SequenceID(1))
			assert.Nil(t, err)
			if spec.ProposalDeleted {
				assert.Equal(t, p.Status, Proposal_Withdrawn)
			} else {
				assert.Equal(t, true, p.Status != Proposal_Withdrawn)
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
		Mods           func(weave.Context, *Proposal)            // modifies test fixtures before storing
		Msg            VoteMsg
		SignedBy       weave.Condition
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
		Exp            TallyResult
		ExpVotedBy     weave.Address
	}{
		"Vote Yes": {
			Msg:        VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			SignedBy:   aliceCond,
			Exp:        TallyResult{TotalYes: 1, Threshold: Fraction{Numerator: 1, Denominator: 2}, TotalElectorateWeight: 11},
			ExpVotedBy: alice,
		},
		"Vote No": {
			Msg:        VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_No, Voter: alice},
			SignedBy:   aliceCond,
			Exp:        TallyResult{TotalNo: 1, Threshold: Fraction{Numerator: 1, Denominator: 2}, TotalElectorateWeight: 11},
			ExpVotedBy: alice,
		},
		"Vote Abstain": {
			Msg:        VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Abstain, Voter: alice},
			SignedBy:   aliceCond,
			Exp:        TallyResult{TotalAbstain: 1, Threshold: Fraction{Numerator: 1, Denominator: 2}, TotalElectorateWeight: 11},
			ExpVotedBy: alice,
		},
		"Vote counts weights": {
			Msg:        VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Abstain, Voter: bobby},
			SignedBy:   bobbyCond,
			Exp:        TallyResult{TotalAbstain: 10, Threshold: Fraction{Numerator: 1, Denominator: 2}, TotalElectorateWeight: 11},
			ExpVotedBy: bobby,
		},
		"Vote defaults to main signer when no voter address submitted": {
			Msg:        VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes},
			SignedBy:   aliceCond,
			Exp:        TallyResult{TotalYes: 1, Threshold: Fraction{Numerator: 1, Denominator: 2}, TotalElectorateWeight: 11},
			ExpVotedBy: alice,
		},
		"Can change vote": {
			Init: func(ctx weave.Context, db store.KVStore) {
				vBucket := NewVoteBucket()
				obj := vBucket.Build(db, proposalID,
					Vote{
						Metadata: &weave.Metadata{Schema: 1},
						Voted:    VoteOption_Yes,
						Elector:  Elector{Address: bobby, Weight: 10},
					},
				)
				vBucket.Save(db, obj)
			},
			Mods: func(ctx weave.Context, proposal *Proposal) {
				proposal.VoteState.TotalYes = 10
			},
			Msg:        VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_No, Voter: bobby},
			SignedBy:   bobbyCond,
			Exp:        TallyResult{TotalNo: 10, TotalYes: 0, Threshold: Fraction{Numerator: 1, Denominator: 2}, TotalElectorateWeight: 11},
			ExpVotedBy: bobby,
		},
		"Can resubmit vote": {
			Init: func(ctx weave.Context, db store.KVStore) {
				vBucket := NewVoteBucket()
				obj := vBucket.Build(db, proposalID,
					Vote{
						Metadata: &weave.Metadata{Schema: 1},
						Voted:    VoteOption_Yes,
						Elector:  Elector{Address: alice, Weight: 1},
					},
				)
				vBucket.Save(db, obj)
			},
			Mods: func(ctx weave.Context, proposal *Proposal) {
				proposal.VoteState.TotalYes = 1
			},
			Msg:        VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			SignedBy:   aliceCond,
			Exp:        TallyResult{TotalYes: 1, Threshold: Fraction{Numerator: 1, Denominator: 2}, TotalElectorateWeight: 11},
			ExpVotedBy: alice,
		},
		"Voter must sign": {
			Msg:            VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes, Voter: bobby},
			SignedBy:       aliceCond,
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
		},
		"Vote with invalid option": {
			Msg:            VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Invalid, Voter: alice},
			SignedBy:       aliceCond,
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"Voter not in electorate must be rejected": {
			Msg:            VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes, Voter: nonElector},
			SignedBy:       nonElectorCond,
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
		},
		"Vote before start date": {
			Mods: func(ctx weave.Context, proposal *Proposal) {
				blockTime, _ := weave.BlockTime(ctx)
				proposal.VotingStartTime = weave.AsUnixTime(blockTime.Add(time.Second))
			},
			Msg:            VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			SignedBy:       aliceCond,
			WantCheckErr:   errors.ErrState,
			WantDeliverErr: errors.ErrState,
		},
		"Vote on start date": {
			Mods: func(ctx weave.Context, proposal *Proposal) {
				blockTime, _ := weave.BlockTime(ctx)
				proposal.VotingStartTime = weave.AsUnixTime(blockTime)
			},
			Msg:            VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			SignedBy:       aliceCond,
			WantCheckErr:   errors.ErrState,
			WantDeliverErr: errors.ErrState,
		},
		"Vote on end date": {
			Mods: func(ctx weave.Context, proposal *Proposal) {
				blockTime, _ := weave.BlockTime(ctx)
				proposal.VotingEndTime = weave.AsUnixTime(blockTime)
			},
			Msg:            VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			SignedBy:       aliceCond,
			WantCheckErr:   errors.ErrState,
			WantDeliverErr: errors.ErrState,
		},
		"Vote after end date": {
			Mods: func(ctx weave.Context, proposal *Proposal) {
				blockTime, _ := weave.BlockTime(ctx)
				proposal.VotingEndTime = weave.AsUnixTime(blockTime.Add(-1 * time.Second))
			},
			Msg:            VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			SignedBy:       aliceCond,
			WantCheckErr:   errors.ErrState,
			WantDeliverErr: errors.ErrState,
		},
		"Vote on withdrawn proposal must fail": {
			Mods: func(ctx weave.Context, proposal *Proposal) {
				proposal.Status = Proposal_Withdrawn
			},
			Msg:            VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			SignedBy:       aliceCond,
			WantCheckErr:   errors.ErrState,
			WantDeliverErr: errors.ErrState,
		},
		"Vote on closed proposal must fail": {
			Mods: func(ctx weave.Context, proposal *Proposal) {
				proposal.Status = Proposal_Closed
			},
			Msg:            VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			SignedBy:       aliceCond,
			WantCheckErr:   errors.ErrState,
			WantDeliverErr: errors.ErrState,
		},
		"Sanity check on count vote": {
			Mods: func(ctx weave.Context, proposal *Proposal) {
				// not a valid setup
				proposal.VoteState.TotalYes = math.MaxUint64
				proposal.VoteState.TotalElectorateWeight = math.MaxUint64
			},
			Msg:            VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes, Voter: alice},
			SignedBy:       aliceCond,
			WantDeliverErr: errors.ErrHuman,
		},
		"Sanity check on undo count vote": {
			Init: func(ctx weave.Context, db store.KVStore) {
				vBucket := NewVoteBucket()
				obj := vBucket.Build(db, proposalID,
					Vote{
						Metadata: &weave.Metadata{Schema: 1},
						Voted:    VoteOption_Yes,
						Elector:  Elector{Address: bobby, Weight: 10},
					},
				)
				vBucket.Save(db, obj)
			},
			Mods: func(ctx weave.Context, proposal *Proposal) {
				// not a valid setup
				proposal.VoteState.TotalYes = 0
				proposal.VoteState.TotalElectorateWeight = math.MaxUint64
			},
			Msg:            VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_No, Voter: bobby},
			SignedBy:       bobbyCond,
			WantDeliverErr: errors.ErrHuman,
		},
	}

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, packageName)

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
			pBucket := withTextProposal(t, db, ctx, spec.Mods)
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
			p, err := pBucket.GetProposal(cache, weavetest.SequenceID(1))
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
		Mods           func(weave.Context, *Proposal)
		Src            tallySetup
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
		ExpResult      Proposal_Result
		PostChecks     func(t *testing.T, kvStore weave.KVStore)
	}{
		"Accepted with electorate majority": {
			Src: tallySetup{
				yes:                   5,
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: Proposal_Accepted,
		},
		"Accepted with all yes votes required": {
			Src: tallySetup{
				yes:                   9,
				threshold:             Fraction{Numerator: 1, Denominator: 1},
				totalWeightElectorate: 9,
			},
			ExpResult: Proposal_Accepted,
		},
		"Rejected without enough Yes votes": {
			Src: tallySetup{
				yes:                   4,
				abstain:               5,
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: Proposal_Rejected,
		},
		"Rejected on acceptance threshold value": {
			Src: tallySetup{
				yes:                   4,
				no:                    1,
				abstain:               3,
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: Proposal_Rejected,
		},
		"Rejected without voters": {
			Src: tallySetup{
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 2,
			},
			ExpResult: Proposal_Rejected,
		},
		"Rejected without enough votes: 2/3": {
			Src: tallySetup{
				yes:                   6,
				threshold:             Fraction{Numerator: 2, Denominator: 3},
				totalWeightElectorate: 9,
			},
			ExpResult: Proposal_Rejected,
		},
		"Accepted with quorum and acceptance thresholds exceeded: 5/9": {
			Src: tallySetup{
				yes:                   5,
				quorum:                &Fraction{Numerator: 1, Denominator: 2},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: Proposal_Accepted,
		},
		"Rejected with quorum thresholds not exceeded": {
			Src: tallySetup{
				yes:                   4,
				quorum:                &Fraction{Numerator: 1, Denominator: 2},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: Proposal_Rejected,
		},
		"Accepted with quorum and acceptance thresholds exceeded: 4/9": {
			Src: tallySetup{
				yes:                   4,
				no:                    1,
				quorum:                &Fraction{Numerator: 1, Denominator: 2},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: Proposal_Accepted,
		},
		"Rejected with majority No": {
			Src: tallySetup{
				yes:                   4,
				no:                    5,
				quorum:                &Fraction{Numerator: 1, Denominator: 2},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: Proposal_Rejected,
		},
		"Rejected by single No when unanimity required": {
			Src: tallySetup{
				yes:                   8,
				no:                    1,
				quorum:                &Fraction{Numerator: 1, Denominator: 2},
				threshold:             Fraction{Numerator: 1, Denominator: 1},
				totalWeightElectorate: 9,
			},
			ExpResult: Proposal_Rejected,
		},
		"Rejected by missing vote when all required": {
			Src: tallySetup{
				yes:                   8,
				quorum:                &Fraction{Numerator: 1, Denominator: 1},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: Proposal_Rejected,
		},
		"Accept on quorum fraction 1/1": {
			Src: tallySetup{
				yes:                   8,
				abstain:               1,
				quorum:                &Fraction{Numerator: 1, Denominator: 1},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: Proposal_Accepted,
		},
		"Accepted with quorum and acceptance thresholds exceeded: 3/9": {
			Src: tallySetup{
				yes:                   3,
				abstain:               2,
				quorum:                &Fraction{Numerator: 1, Denominator: 2},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: Proposal_Accepted,
		},
		"Accepted by single Yes and neutral abstains": {
			Src: tallySetup{
				yes:                   1,
				abstain:               4,
				quorum:                &Fraction{Numerator: 1, Denominator: 2},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: Proposal_Accepted,
		},
		"Rejected without Yes majority and neutral abstains": {
			Src: tallySetup{
				yes:                   2,
				no:                    2,
				abstain:               2,
				quorum:                &Fraction{Numerator: 1, Denominator: 2},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: Proposal_Rejected,
		},
		"Accepted with acceptance thresholds < quorum": {
			Src: tallySetup{
				yes:                   2,
				abstain:               5,
				quorum:                &Fraction{Numerator: 2, Denominator: 3},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult: Proposal_Accepted,
		},
		"Accepted with quorum and acceptance thresholds require all votes": {
			Src: tallySetup{
				yes:                   9,
				quorum:                &Fraction{Numerator: 1, Denominator: 1},
				threshold:             Fraction{Numerator: 1, Denominator: 1},
				totalWeightElectorate: 9,
			},
			ExpResult: Proposal_Accepted,
		},
		"Works with high values: accept": {
			Src: tallySetup{
				yes:                   math.MaxUint64,
				no:                    math.MaxUint64/2 - 1, // less than 1/2 yes
				abstain:               math.MaxUint64,
				quorum:                &Fraction{Numerator: math.MaxUint32 - 1, Denominator: math.MaxUint32},
				threshold:             Fraction{Numerator: math.MaxUint32 - 1, Denominator: math.MaxUint32},
				totalWeightElectorate: math.MaxUint64,
			},
			ExpResult: Proposal_Accepted,
		},
		"Works with high values: reject": {
			Src: tallySetup{
				yes:                   math.MaxUint64 - 1, // less than total electorate
				no:                    math.MaxUint64 - 1,
				abstain:               math.MaxUint64,
				quorum:                &Fraction{Numerator: math.MaxUint32 - 1, Denominator: math.MaxUint32},
				threshold:             Fraction{Numerator: math.MaxUint32 - 1, Denominator: math.MaxUint32},
				totalWeightElectorate: math.MaxUint64,
			},
			ExpResult: Proposal_Rejected,
		},
		"Updates an electorate on success": {
			Mods: func(ctx weave.Context, p *Proposal) {
				update := updateElectoreateProposalFixture()
				blockTime, _ := weave.BlockTime(ctx)
				update.VotingEndTime = weave.AsUnixTime(blockTime.Add(-1 * time.Second))
				update.VoteState = p.VoteState
				*p = update
			},
			Src: tallySetup{
				yes:                   10,
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 11,
			},
			ExpResult: Proposal_Accepted,
			PostChecks: func(t *testing.T, db weave.KVStore) {
				elect, err := NewElectorateBucket().GetElectorate(db, weavetest.SequenceID(1))
				if err != nil {
					t.Fatalf("unexpected error: %+v", err)
				}
				got := elect.Electors
				exp := []Elector{{alice, 10}, {bobby, 10}}
				sortByAddress(exp)
				if !reflect.DeepEqual(exp, got) {
					t.Errorf("expected %v but got %v", exp, got)
				}
				if exp, got := uint64(20), elect.TotalElectorateWeight; exp != got {
					t.Errorf("expected %v but got %v", exp, got)
				}
			},
		},
		"Does not update an electorate when rejected": {
			Mods: func(ctx weave.Context, p *Proposal) {
				update := updateElectoreateProposalFixture()
				blockTime, _ := weave.BlockTime(ctx)
				update.VotingEndTime = weave.AsUnixTime(blockTime.Add(-1 * time.Second))
				update.VoteState = p.VoteState
				*p = update
			},
			Src: tallySetup{
				yes:                   1,
				no:                    10,
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 11,
			},
			ExpResult: Proposal_Rejected,
			PostChecks: func(t *testing.T, db weave.KVStore) {
				elect, err := NewElectorateBucket().GetElectorate(db, weavetest.SequenceID(1))
				if err != nil {
					t.Fatalf("unexpected error: %+v", err)
				}
				got := elect.Electors
				exp := []Elector{{alice, 1}, {bobby, 10}}
				sortByAddress(exp)
				if !reflect.DeepEqual(exp, got) {
					t.Errorf("expected %v but got %v", exp, got)
				}
				if exp, got := uint64(11), elect.TotalElectorateWeight; exp != got {
					t.Errorf("expected %v but got %v", exp, got)
				}
			},
		},
		"Fails on second tally": {
			Mods: func(_ weave.Context, p *Proposal) {
				p.Status = Proposal_Closed
			},
			Src: tallySetup{
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 11,
			},
			WantCheckErr:   errors.ErrState,
			WantDeliverErr: errors.ErrState,
			ExpResult:      Proposal_Accepted,
		},
		"Fails on tally before end date": {
			Mods: func(ctx weave.Context, p *Proposal) {
				blockTime, _ := weave.BlockTime(ctx)
				p.VotingEndTime = weave.AsUnixTime(blockTime.Add(time.Second))
			},
			Src: tallySetup{
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 11,
			},
			WantCheckErr:   errors.ErrState,
			WantDeliverErr: errors.ErrState,
			ExpResult:      Proposal_Undefined,
		},
		"Fails on tally at end date": {
			Mods: func(ctx weave.Context, p *Proposal) {
				blockTime, _ := weave.BlockTime(ctx)
				p.VotingEndTime = weave.AsUnixTime(blockTime)
			},
			Src: tallySetup{
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 11,
			},
			WantCheckErr:   errors.ErrState,
			WantDeliverErr: errors.ErrState,
			ExpResult:      Proposal_Undefined,
		},
		"Fails on withdrawn proposal": {
			Mods: func(ctx weave.Context, p *Proposal) {
				p.Status = Proposal_Withdrawn
			},
			Src: tallySetup{
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 11,
			},
			WantCheckErr:   errors.ErrState,
			WantDeliverErr: errors.ErrState,
			ExpResult:      Proposal_Undefined,
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
			migration.MustInitPkg(db, packageName)

			// given
			ctx := weave.WithBlockTime(context.Background(), time.Now().Round(time.Second))
			setupForTally := func(_ weave.Context, p *Proposal) {
				p.VoteState = NewTallyResult(spec.Src.quorum, spec.Src.threshold, spec.Src.totalWeightElectorate)
				p.VoteState.TotalYes = spec.Src.yes
				p.VoteState.TotalNo = spec.Src.no
				p.VoteState.TotalAbstain = spec.Src.abstain
				blockTime, _ := weave.BlockTime(ctx)
				p.VotingEndTime = weave.AsUnixTime(blockTime.Add(-1 * time.Second))
			}
			pBucket := withTextProposal(t, db, ctx, append([]ctxAwareMutator{setupForTally}, spec.Mods)...)
			cache := db.CacheWrap()

			// when check is called
			tx := &weavetest.Tx{Msg: &TallyMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: weavetest.SequenceID(1)}}
			if _, err := rt.Check(ctx, cache, tx); !spec.WantCheckErr.Is(err) {
				t.Fatalf("check expected: %+v  but got %+v", spec.WantCheckErr, err)
			}

			cache.Discard()

			// and when deliver is called
			_, err := rt.Deliver(ctx, db, tx)
			if !spec.WantDeliverErr.Is(err) {
				t.Fatalf("deliver expected: %+v  but got %+v", spec.WantCheckErr, err)
			}
			if spec.WantDeliverErr != nil {
				return // skip further checks on expected error
			}
			// and check persisted result
			p, err := pBucket.GetProposal(cache, weavetest.SequenceID(1))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if exp, got := spec.ExpResult, p.Result; exp != got {
				t.Errorf("expected %v but got %v: vote state: %#v", exp, got, p.VoteState)
			}
			if exp, got := Proposal_Closed, p.Status; exp != got {
				t.Errorf("expected %v but got %v", exp, got)
			}
			if spec.PostChecks != nil {
				spec.PostChecks(t, cache)
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
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: electorateID,
				DiffElectors: []Elector{{Address: alice, Weight: 22}},
			},
			SignedBy: bobbyCond,
			ExpModel: &Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Admin:                 bobby,
				Title:                 "fooo",
				Electors:              []Elector{{Address: alice, Weight: 22}, {Address: bobby, Weight: 10}},
				TotalElectorateWeight: 32,
				UpdateElectionRuleID:  weavetest.SequenceID(1),
			},
		},
		"Update to remove address": {
			Msg: UpdateElectorateMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: electorateID,
				DiffElectors: []Elector{{Address: alice, Weight: 0}},
			},
			SignedBy: bobbyCond,
			ExpModel: &Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Admin:                 bobby,
				Title:                 "fooo",
				Electors:              []Elector{{Address: bobby, Weight: 10}},
				TotalElectorateWeight: 10,
				UpdateElectionRuleID:  weavetest.SequenceID(1),
			},
		},
		"Update to add a new address": {
			Msg: UpdateElectorateMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: electorateID,
				DiffElectors: []Elector{{Address: charlie, Weight: 2}},
			},
			SignedBy: bobbyCond,
			ExpModel: &Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Admin:                 bobby,
				Title:                 "fooo",
				Electors:              []Elector{{Address: alice, Weight: 1}, {Address: bobby, Weight: 10}, {Address: charlie, Weight: 2}},
				TotalElectorateWeight: 13,
				UpdateElectionRuleID:  weavetest.SequenceID(1),
			},
		},
		"Update by non owner should fail": {
			Msg: UpdateElectorateMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: electorateID,
				DiffElectors: []Elector{{Address: alice, Weight: 22}},
			},
			SignedBy:       aliceCond,
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
		},
		"Update with open proposal should fail": {
			Msg: UpdateElectorateMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: electorateID,
				DiffElectors: []Elector{{Address: alice, Weight: 22}},
			},
			SignedBy: bobbyCond,
			ExpModel: &Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Admin:                 bobby,
				Title:                 "fooo",
				Electors:              []Elector{{Address: alice, Weight: 22}},
				TotalElectorateWeight: 22,
				UpdateElectionRuleID:  weavetest.SequenceID(1),
			},
			WithProposal:   true,
			WantDeliverErr: errors.ErrState,
		},
		"Update with closed proposal should succeed": {
			Msg: UpdateElectorateMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: electorateID,
				DiffElectors: []Elector{{Address: alice, Weight: 22}, {Address: bobby}, {Address: charlie, Weight: 2}},
			},
			SignedBy: bobbyCond,
			ExpModel: &Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Admin:                 bobby,
				Title:                 "fooo",
				Electors:              []Elector{{Address: alice, Weight: 22}, {Address: charlie, Weight: 2}},
				TotalElectorateWeight: 24,
				UpdateElectionRuleID:  weavetest.SequenceID(1),
			},
			WithProposal: true,
			Mods: func(ctx weave.Context, proposal *Proposal) {
				proposal.Status = Proposal_Closed
			},
		},
		"Update with too many electors should fail": {
			Msg: UpdateElectorateMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: electorateID,
				DiffElectors: buildElectors(2001),
			},
			SignedBy:       bobbyCond,
			WantDeliverErr: errors.ErrInput,
		},
		"Update without electors should fail": {
			Msg: UpdateElectorateMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: electorateID,
			},
			SignedBy:       bobbyCond,
			WantCheckErr:   errors.ErrEmpty,
			WantDeliverErr: errors.ErrEmpty,
		},
		"Duplicate electors should fail": {
			Msg: UpdateElectorateMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: electorateID,
				DiffElectors: []Elector{{Address: alice, Weight: 1}, {Address: alice, Weight: 2}},
			},
			SignedBy:       bobbyCond,
			WantCheckErr:   errors.ErrDuplicate,
			WantDeliverErr: errors.ErrDuplicate,
		},
		"Empty address in electors should fail": {
			Msg: UpdateElectorateMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: electorateID,
				DiffElectors: []Elector{{Address: weave.Address{}, Weight: 1}},
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
			migration.MustInitPkg(db, packageName)

			withElectorate(t, db)
			if spec.WithProposal {
				withTextProposal(t, db, nil, spec.Mods)
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
			sortByAddress(spec.ExpModel.Electors)
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
				Metadata:          &weave.Metadata{Schema: 1},
				ElectionRuleID:    electionRulesID,
				VotingPeriodHours: 12,
				Threshold:         Fraction{Numerator: 2, Denominator: 3},
			},
			SignedBy: bobbyCond,
			ExpModel: &ElectionRule{
				Metadata:          &weave.Metadata{Schema: 1},
				Admin:             bobby,
				Title:             "barr",
				VotingPeriodHours: 12,
				Threshold:         Fraction{Numerator: 2, Denominator: 3},
			},
		},
		"Update with max voting time": {
			Msg: UpdateElectionRuleMsg{
				Metadata:          &weave.Metadata{Schema: 1},
				ElectionRuleID:    electionRulesID,
				VotingPeriodHours: 4 * 7 * 24,
				Threshold:         Fraction{Numerator: 2, Denominator: 3},
			},
			SignedBy: bobbyCond,
			ExpModel: &ElectionRule{
				Metadata:          &weave.Metadata{Schema: 1},
				Admin:             bobby,
				Title:             "barr",
				VotingPeriodHours: 4 * 7 * 24,
				Threshold:         Fraction{Numerator: 2, Denominator: 3},
			},
		},
		"Update by non owner should fail": {
			Msg: UpdateElectionRuleMsg{
				Metadata:          &weave.Metadata{Schema: 1},
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
				Metadata:          &weave.Metadata{Schema: 1},
				ElectionRuleID:    electionRulesID,
				VotingPeriodHours: 12,
				Threshold:         Fraction{Numerator: 3, Denominator: 2},
			},
			SignedBy:       bobbyCond,
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"voting period hours must not be empty": {
			Msg: UpdateElectionRuleMsg{
				Metadata:          &weave.Metadata{Schema: 1},
				ElectionRuleID:    electionRulesID,
				VotingPeriodHours: 0,
				Threshold:         Fraction{Numerator: 1, Denominator: 2},
			},
			SignedBy:       bobbyCond,
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"voting period hours must not exceed max": {
			Msg: UpdateElectionRuleMsg{
				Metadata:          &weave.Metadata{Schema: 1},
				ElectionRuleID:    electionRulesID,
				VotingPeriodHours: 4*7*24 + 1,
				Threshold:         Fraction{Numerator: 1, Denominator: 2},
			},
			SignedBy:       bobbyCond,
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
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
			migration.MustInitPkg(db, packageName)

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
	proposal := textProposalFixture(ctxMods...)
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
		Admin:    bobby,
		Electors: []Elector{
			{Address: alice, Weight: 1},
			{Address: bobby, Weight: 10},
		},
		TotalElectorateWeight: 11,
		UpdateElectionRuleID:  weavetest.SequenceID(1),
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
		Admin:             bobby,
		VotingPeriodHours: 1,
		Threshold:         Fraction{1, 2},
	}

	if _, err := rulesBucket.Create(db, rule); err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	return rule
}
