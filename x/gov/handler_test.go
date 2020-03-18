package gov

import (
	"context"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

var (
	hAliceCond   = weavetest.NewCondition()
	hAlice       = hAliceCond.Address()
	hBobbyCond   = weavetest.NewCondition()
	hBobby       = hBobbyCond.Address()
	hCharlieCond = weavetest.NewCondition()
	hCharlie     = hCharlieCond.Address()
)

func TestCreateTextProposal(t *testing.T) {
	now := weave.AsUnixTime(time.Now())

	textOption, electorateOption, ruleOption := genTextOptions(t), genElectorateOptions(t), genRuleOptions(t)
	invalidOption, garbageOption := generateInvalidOptions(t)

	specs := map[string]struct {
		Init           func(t *testing.T, db store.KVStore)
		Msg            CreateProposalMsg
		Signers        []weave.Condition
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
		Exp            Proposal
		ExpProposer    weave.Address
	}{
		"Happy path with text option": {
			Msg: CreateProposalMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now.Add(time.Hour),
				ElectionRuleID: weavetest.SequenceID(1),
				Author:         hBobby,
				RawOption:      textOption,
			},
			Signers: []weave.Condition{hAliceCond, hBobbyCond},
			Exp: Proposal{
				Metadata:        &weave.Metadata{Schema: 1},
				Title:           "my proposal",
				Description:     "my description",
				ElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
				ElectorateRef:   orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
				VotingStartTime: now.Add(time.Hour),
				VotingEndTime:   now.Add(2 * time.Hour),
				Status:          Proposal_Submitted,
				Result:          Proposal_Undefined,
				ExecutorResult:  Proposal_NotRun,
				SubmissionTime:  now,
				Author:          hBobby,
				VoteState: TallyResult{
					Threshold:             Fraction{Numerator: 1, Denominator: 2},
					TotalElectorateWeight: 11,
				},
				RawOption: textOption,
			},
			ExpProposer: hBobby,
		},
		"Happy path with electorate option": {
			Msg: CreateProposalMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				Title:          "new electorate",
				Description:    "a very good readon",
				StartTime:      now.Add(time.Hour),
				ElectionRuleID: weavetest.SequenceID(1),
				Author:         hBobby,
				RawOption:      electorateOption,
			},
			Signers: []weave.Condition{hAliceCond, hBobbyCond},
			Exp: Proposal{
				Metadata:        &weave.Metadata{Schema: 1},
				Title:           "new electorate",
				Description:     "a very good readon",
				ElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
				ElectorateRef:   orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
				VotingStartTime: now.Add(time.Hour),
				VotingEndTime:   now.Add(2 * time.Hour),
				Status:          Proposal_Submitted,
				Result:          Proposal_Undefined,
				ExecutorResult:  Proposal_NotRun,
				SubmissionTime:  now,
				Author:          hBobby,
				VoteState: TallyResult{
					Threshold:             Fraction{Numerator: 1, Denominator: 2},
					TotalElectorateWeight: 11,
				},
				RawOption: electorateOption,
			},
			ExpProposer: hBobby,
		},
		"Happy path with election rule option": {
			Msg: CreateProposalMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				Title:          "new rule",
				Description:    "a very good readon",
				StartTime:      now.Add(time.Hour),
				ElectionRuleID: weavetest.SequenceID(1),
				Author:         hBobby,
				RawOption:      ruleOption,
			},
			Signers: []weave.Condition{hAliceCond, hBobbyCond},
			Exp: Proposal{
				Metadata:        &weave.Metadata{Schema: 1},
				Title:           "new rule",
				Description:     "a very good readon",
				ElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
				ElectorateRef:   orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
				VotingStartTime: now.Add(time.Hour),
				VotingEndTime:   now.Add(2 * time.Hour),
				Status:          Proposal_Submitted,
				Result:          Proposal_Undefined,
				ExecutorResult:  Proposal_NotRun,
				SubmissionTime:  now,
				Author:          hBobby,
				VoteState: TallyResult{
					Threshold:             Fraction{Numerator: 1, Denominator: 2},
					TotalElectorateWeight: 11,
				},
				RawOption: ruleOption,
			},
			ExpProposer: hBobby,
		},
		"All good with main signer as author": {
			Msg: CreateProposalMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now.Add(time.Hour),
				ElectionRuleID: weavetest.SequenceID(1),
				RawOption:      textOption,
			},
			Signers: []weave.Condition{hAliceCond, hBobbyCond},
			Exp: Proposal{
				Metadata:        &weave.Metadata{Schema: 1},
				Title:           "my proposal",
				Description:     "my description",
				ElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
				ElectorateRef:   orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
				VotingStartTime: now.Add(time.Hour),
				VotingEndTime:   now.Add(2 * time.Hour),
				Status:          Proposal_Submitted,
				Result:          Proposal_Undefined,
				ExecutorResult:  Proposal_NotRun,
				SubmissionTime:  now,
				Author:          hAlice,
				VoteState: TallyResult{
					Threshold:             Fraction{Numerator: 1, Denominator: 2},
					TotalElectorateWeight: 11,
				},
				RawOption: textOption,
			},
			ExpProposer: hAlice,
		},
		"Invalid Option": {
			Msg: CreateProposalMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now.Add(time.Hour),
				ElectionRuleID: weavetest.SequenceID(1),
				Author:         hBobby,
				RawOption:      invalidOption,
			},
			Signers:        []weave.Condition{hAliceCond, hBobbyCond},
			ExpProposer:    hBobby,
			WantCheckErr:   errors.ErrEmpty,
			WantDeliverErr: errors.ErrEmpty,
		},
		"Cannot Decode Option": {
			Msg: CreateProposalMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now.Add(time.Hour),
				ElectionRuleID: weavetest.SequenceID(1),
				Author:         hBobby,
				RawOption:      garbageOption,
			},
			Signers:        []weave.Condition{hAliceCond, hBobbyCond},
			ExpProposer:    hBobby,
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"ElectionRuleID missing": {
			Msg: CreateProposalMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				Title:       "my proposal",
				Description: "my description",
				StartTime:   now.Add(time.Hour),
				Author:      hBobby,
				RawOption:   textOption,
			},
			Signers:        []weave.Condition{hAliceCond, hBobbyCond},
			ExpProposer:    hBobby,
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"ElectionRuleID invalid": {
			Msg: CreateProposalMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now.Add(time.Hour),
				ElectionRuleID: weavetest.SequenceID(10000),
				Author:         hBobby,
				RawOption:      textOption,
			},
			Signers:        []weave.Condition{hAliceCond, hBobbyCond},
			ExpProposer:    hBobby,
			WantCheckErr:   errors.ErrNotFound,
			WantDeliverErr: errors.ErrNotFound,
		},
		"Author has not signed so message should be rejected": {
			Msg: CreateProposalMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now.Add(time.Hour),
				ElectionRuleID: weavetest.SequenceID(1),
				Author:         weavetest.NewCondition().Address(),
				RawOption:      textOption,
			},
			Signers:        []weave.Condition{hAliceCond, hBobbyCond},
			ExpProposer:    hBobby,
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
		},
		"Start time not in the future": {
			Msg: CreateProposalMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now,
				ElectionRuleID: weavetest.SequenceID(1),
				Author:         hBobby,
				RawOption:      textOption,
			},
			Signers:        []weave.Condition{hAliceCond, hBobbyCond},
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"Start time too far in the future": {
			Msg: CreateProposalMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now.Add(7*24*time.Hour + time.Second),
				ElectionRuleID: weavetest.SequenceID(1),
				Author:         hBobby,
				RawOption:      textOption,
			},
			Signers:        []weave.Condition{hAliceCond, hBobbyCond},
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"A proposal creation is restricted to electorate members only": {
			Init: func(t *testing.T, db weave.KVStore) {
				createElectorate(t, db, []weave.Address{
					hAlice,
				})
				createElectorate(t, db, []weave.Address{
					hBobby,
					hCharlie,
				})
			},
			Msg: CreateProposalMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now.Add(time.Hour),
				ElectionRuleID: weavetest.SequenceID(1),
				Author:         hBobby,
				RawOption:      textOption,
			},
			// Both signers are from the second electorate while
			// the proposal is created for the first one.
			Signers:        []weave.Condition{hBobbyCond, hCharlieCond},
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
		},
		"A proposal creation can be signed by any electorate member": {
			Init: func(t *testing.T, db weave.KVStore) {
				createElectorate(t, db, []weave.Address{
					hAlice,
					hCharlie,
				})
				createElectorate(t, db, []weave.Address{
					hBobby,
					hCharlie,
				})
			},
			Msg: CreateProposalMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				Title:          "my proposal",
				Description:    "my description",
				StartTime:      now.Add(time.Hour),
				ElectionRuleID: weavetest.SequenceID(1),
				Author:         hBobby,
				RawOption:      textOption,
			},
			// Both signers are from the second electorate while
			// the proposal is created for the first one.
			Signers:        []weave.Condition{hBobbyCond, hCharlieCond},
			WantCheckErr:   nil,
			WantDeliverErr: nil,
			Exp: Proposal{
				Metadata:        &weave.Metadata{Schema: 1},
				Title:           "my proposal",
				Description:     "my description",
				ElectionRuleRef: orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
				ElectorateRef:   orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
				VotingStartTime: now.Add(time.Hour),
				VotingEndTime:   now.Add(2 * time.Hour),
				Status:          Proposal_Submitted,
				Result:          Proposal_Undefined,
				ExecutorResult:  Proposal_NotRun,
				SubmissionTime:  now,
				Author:          hBobby,
				VoteState: TallyResult{
					Threshold:             Fraction{Numerator: 1, Denominator: 2},
					TotalElectorateWeight: 3,
				},
				RawOption: textOption,
			},
			ExpProposer: hBobby,
		},
	}

	for testName, spec := range specs {
		t.Run(testName, func(t *testing.T) {
			auth := &weavetest.Auth{
				Signers: spec.Signers,
			}
			rt := app.NewRouter()
			cron := &weavetest.Cron{}
			// We don't run the executor here, so we can safely pass in nil.
			RegisterRoutes(rt, auth, decodeProposalOptions, nil, cron)

			db := store.MemStore()
			migration.MustInitPkg(db, packageName)

			// given
			ctx := weave.WithBlockTime(context.Background(), now.Time())
			if spec.Init != nil {
				spec.Init(t, db)
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

			assertProposalsEqual(t, spec.Exp, *p)
			cache.Discard()
		})
	}
}

func assertProposalsEqual(t testing.TB, a, b Proposal) {
	t.Helper()

	// TallyTaskID is a random value that we do not care about.
	a.TallyTaskID = nil
	b.TallyTaskID = nil

	if !reflect.DeepEqual(a, b) {
		t.Logf("a: %#v", a)
		t.Logf("b: %#v", b)
		t.Fatal("unexpected proposal state")
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
			Msg:             DeleteProposalMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID},
			SignedBy:        hAliceCond,
			ProposalDeleted: true,
			Mods: func(ctx weave.Context, proposal *Proposal) {
				proposal.VotingStartTime = weave.AsUnixTime(time.Now().Add(1 * time.Hour))
				proposal.VotingEndTime = weave.AsUnixTime(time.Now().Add(2 * time.Hour))
			},
		},
		"Proposal does not exist": {
			Msg:            DeleteProposalMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: nonExistentProposalID},
			SignedBy:       hAliceCond,
			WantCheckErr:   errors.ErrNotFound,
			WantDeliverErr: errors.ErrNotFound,
		},
		"Delete by non-author": {
			Msg:            DeleteProposalMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID},
			SignedBy:       hBobbyCond,
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
			Mods: func(ctx weave.Context, proposal *Proposal) {
				proposal.VotingStartTime = weave.AsUnixTime(time.Now().Add(1 * time.Hour))
				proposal.VotingEndTime = weave.AsUnixTime(time.Now().Add(2 * time.Hour))
			},
		},
		"Voting has started": {
			Msg:      DeleteProposalMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID},
			SignedBy: hAliceCond,
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
			RegisterRoutes(rt, auth, decodeProposalOptions, nil, &weavetest.Cron{})

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

func TestCreateResolution(t *testing.T) {
	ID := weavetest.SequenceID(1)
	proposal := Proposal{ElectorateRef: orm.VersionedIDRef{ID: ID, Version: 1}}

	specs := map[string]struct {
		ctx            weave.Context
		Msg            CreateTextResolutionMsg
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
		created        bool
	}{
		"Happy path": {
			Msg:     CreateTextResolutionMsg{Metadata: &weave.Metadata{Schema: 1}, Resolution: "123"},
			ctx:     withProposal(context.Background(), &proposal, ID),
			created: true,
		},
		"Proposal not in context": {
			Msg:            CreateTextResolutionMsg{Metadata: &weave.Metadata{Schema: 1}, Resolution: "123"},
			ctx:            context.Background(),
			WantDeliverErr: errors.ErrNotFound,
		},
		"Invalid Resolution": {
			Msg:            CreateTextResolutionMsg{Metadata: &weave.Metadata{Schema: 1}, Resolution: ""},
			ctx:            withProposal(context.Background(), &proposal, ID),
			WantDeliverErr: errors.ErrEmpty,
			WantCheckErr:   errors.ErrEmpty,
		},
	}

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, packageName)

			auth := &weavetest.Auth{}
			rt := app.NewRouter()
			RegisterBasicProposalRouters(rt, auth)

			// given
			bucket := NewResolutionBucket()
			cache := db.CacheWrap()

			// when check
			tx := &weavetest.Tx{Msg: &spec.Msg}
			if _, err := rt.Check(spec.ctx, cache, tx); !spec.WantCheckErr.Is(err) {
				t.Fatalf("check expected: %+v  but got %+v", spec.WantCheckErr, err)
			}

			cache.Discard()
			// and when deliver
			if _, err := rt.Deliver(spec.ctx, db, tx); !spec.WantDeliverErr.Is(err) {
				t.Fatalf("deliver expected: %+v  but got %+v", spec.WantCheckErr, err)
			}

			if spec.WantDeliverErr != nil {
				return // skip further checks on expected error
			}

			// check that resolution gets created
			r, err := bucket.GetResolution(cache, weavetest.SequenceID(1))
			assert.Nil(t, err)
			if spec.created {
				assert.Equal(t, r != nil, true)
			} else {
				assert.Equal(t, r == nil, true)
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
			Msg:        VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes, Voter: hAlice},
			SignedBy:   hAliceCond,
			Exp:        TallyResult{TotalYes: 1, Threshold: Fraction{Numerator: 1, Denominator: 2}, TotalElectorateWeight: 11},
			ExpVotedBy: hAlice,
		},
		"Vote No": {
			Msg:        VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_No, Voter: hAlice},
			SignedBy:   hAliceCond,
			Exp:        TallyResult{TotalNo: 1, Threshold: Fraction{Numerator: 1, Denominator: 2}, TotalElectorateWeight: 11},
			ExpVotedBy: hAlice,
		},
		"Vote Abstain": {
			Msg:        VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Abstain, Voter: hAlice},
			SignedBy:   hAliceCond,
			Exp:        TallyResult{TotalAbstain: 1, Threshold: Fraction{Numerator: 1, Denominator: 2}, TotalElectorateWeight: 11},
			ExpVotedBy: hAlice,
		},
		"Vote counts weights": {
			Msg:        VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Abstain, Voter: hBobby},
			SignedBy:   hBobbyCond,
			Exp:        TallyResult{TotalAbstain: 10, Threshold: Fraction{Numerator: 1, Denominator: 2}, TotalElectorateWeight: 11},
			ExpVotedBy: hBobby,
		},
		"Vote defaults to main signer when no voter address submitted": {
			Msg:        VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes},
			SignedBy:   hAliceCond,
			Exp:        TallyResult{TotalYes: 1, Threshold: Fraction{Numerator: 1, Denominator: 2}, TotalElectorateWeight: 11},
			ExpVotedBy: hAlice,
		},
		"Can change vote": {
			Init: func(ctx weave.Context, db store.KVStore) {
				vBucket := NewVoteBucket()
				obj := vBucket.Build(db, proposalID,
					Vote{
						Metadata: &weave.Metadata{Schema: 1},
						Voted:    VoteOption_Yes,
						Elector:  Elector{Address: hBobby, Weight: 10},
					},
				)
				vBucket.Save(db, obj)
			},
			Mods: func(ctx weave.Context, proposal *Proposal) {
				proposal.VoteState.TotalYes = 10
			},
			Msg:        VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_No, Voter: hBobby},
			SignedBy:   hBobbyCond,
			Exp:        TallyResult{TotalNo: 10, TotalYes: 0, Threshold: Fraction{Numerator: 1, Denominator: 2}, TotalElectorateWeight: 11},
			ExpVotedBy: hBobby,
		},
		"Can resubmit vote": {
			Init: func(ctx weave.Context, db store.KVStore) {
				vBucket := NewVoteBucket()
				obj := vBucket.Build(db, proposalID,
					Vote{
						Metadata: &weave.Metadata{Schema: 1},
						Voted:    VoteOption_Yes,
						Elector:  Elector{Address: hAlice, Weight: 1},
					},
				)
				vBucket.Save(db, obj)
			},
			Mods: func(ctx weave.Context, proposal *Proposal) {
				proposal.VoteState.TotalYes = 1
			},
			Msg:        VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes, Voter: hAlice},
			SignedBy:   hAliceCond,
			Exp:        TallyResult{TotalYes: 1, Threshold: Fraction{Numerator: 1, Denominator: 2}, TotalElectorateWeight: 11},
			ExpVotedBy: hAlice,
		},
		"Voter must sign": {
			Msg:            VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes, Voter: hBobby},
			SignedBy:       hAliceCond,
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
		},
		"Vote with invalid option": {
			Msg:            VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Invalid, Voter: hAlice},
			SignedBy:       hAliceCond,
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
				proposal.VotingStartTime = unixBlockTime(t, ctx) + 1
			},
			Msg:            VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes, Voter: hAlice},
			SignedBy:       hAliceCond,
			WantCheckErr:   errors.ErrState,
			WantDeliverErr: errors.ErrState,
		},
		"Vote on start date": {
			Mods: func(ctx weave.Context, proposal *Proposal) {
				proposal.VotingStartTime = unixBlockTime(t, ctx)
			},
			Msg:            VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes, Voter: hAlice},
			SignedBy:       hAliceCond,
			WantCheckErr:   errors.ErrState,
			WantDeliverErr: errors.ErrState,
		},
		"Vote on end date": {
			Mods: func(ctx weave.Context, proposal *Proposal) {
				proposal.VotingEndTime = unixBlockTime(t, ctx)
			},
			Msg:            VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes, Voter: hAlice},
			SignedBy:       hAliceCond,
			WantCheckErr:   errors.ErrState,
			WantDeliverErr: errors.ErrState,
		},
		"Vote after end date": {
			Mods: func(ctx weave.Context, proposal *Proposal) {
				proposal.VotingEndTime = unixBlockTime(t, ctx) - 1
			},
			Msg:            VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes, Voter: hAlice},
			SignedBy:       hAliceCond,
			WantCheckErr:   errors.ErrState,
			WantDeliverErr: errors.ErrState,
		},
		"Vote on withdrawn proposal must fail": {
			Mods: func(ctx weave.Context, proposal *Proposal) {
				proposal.Status = Proposal_Withdrawn
			},
			Msg:            VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes, Voter: hAlice},
			SignedBy:       hAliceCond,
			WantCheckErr:   errors.ErrState,
			WantDeliverErr: errors.ErrState,
		},
		"Vote on closed proposal must fail": {
			Mods: func(ctx weave.Context, proposal *Proposal) {
				proposal.Status = Proposal_Closed
			},
			Msg:            VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes, Voter: hAlice},
			SignedBy:       hAliceCond,
			WantCheckErr:   errors.ErrState,
			WantDeliverErr: errors.ErrState,
		},
		"Sanity check on count vote": {
			Mods: func(ctx weave.Context, proposal *Proposal) {
				// not a valid setup
				proposal.VoteState.TotalYes = math.MaxUint64
				proposal.VoteState.TotalElectorateWeight = math.MaxUint64
			},
			Msg:            VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_Yes, Voter: hAlice},
			SignedBy:       hAliceCond,
			WantDeliverErr: errors.ErrHuman,
		},
		"Sanity check on undo count vote": {
			Init: func(ctx weave.Context, db store.KVStore) {
				vBucket := NewVoteBucket()
				obj := vBucket.Build(db, proposalID,
					Vote{
						Metadata: &weave.Metadata{Schema: 1},
						Voted:    VoteOption_Yes,
						Elector:  Elector{Address: hBobby, Weight: 10},
					},
				)
				vBucket.Save(db, obj)
			},
			Mods: func(ctx weave.Context, proposal *Proposal) {
				// not a valid setup
				proposal.VoteState.TotalYes = 0
				proposal.VoteState.TotalElectorateWeight = math.MaxUint64
			},
			Msg:            VoteMsg{Metadata: &weave.Metadata{Schema: 1}, ProposalID: proposalID, Selected: VoteOption_No, Voter: hBobby},
			SignedBy:       hBobbyCond,
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
			RegisterRoutes(rt, auth, decodeProposalOptions, nil, &weavetest.Cron{})

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
		Mods              func(weave.Context, *Proposal)
		Src               tallySetup
		WantDeliverErr    *errors.Error
		WantDeliverLog    string
		ExpResult         Proposal_Result
		ExpExecutorResult Proposal_ExecutorResult
		PostChecks        func(t *testing.T, db weave.KVStore)
		Init              func(t *testing.T, db weave.KVStore)
	}{
		"Accepted with electorate majority": {
			Src: tallySetup{
				yes:                   5,
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult:         Proposal_Accepted,
			ExpExecutorResult: Proposal_Success,
			WantDeliverLog:    "Proposal accepted: execution success",
			PostChecks: func(t *testing.T, db weave.KVStore) {
				obj, err := NewResolutionBucket().Get(db, weavetest.SequenceID(1))
				assert.Nil(t, err)
				res, err := asResolution(obj)
				assert.Nil(t, err)
				assert.Equal(t, res.ElectorateRef, orm.VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1})
				assert.Equal(t, res.Resolution, fixtureResolution)
			},
		},
		"Accepted with all yes votes required": {
			Src: tallySetup{
				yes:                   9,
				threshold:             Fraction{Numerator: 1, Denominator: 1},
				totalWeightElectorate: 9,
			},
			ExpResult:         Proposal_Accepted,
			ExpExecutorResult: Proposal_Success,
			WantDeliverLog:    "Proposal accepted: execution success",
		},
		"Rejected without enough Yes votes": {
			Src: tallySetup{
				yes:                   4,
				abstain:               5,
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult:         Proposal_Rejected,
			ExpExecutorResult: Proposal_NotRun,
			WantDeliverLog:    "Proposal not accepted",
			PostChecks: func(t *testing.T, db weave.KVStore) {
				obj, err := NewResolutionBucket().Get(db, weavetest.SequenceID(1))
				assert.Nil(t, obj)
				// NotFound objects return nil, nil (why not errors.ErrNotFound??)
				assert.Nil(t, err)
			},
		},
		"Rejected on acceptance threshold value": {
			Src: tallySetup{
				yes:                   4,
				no:                    1,
				abstain:               3,
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult:         Proposal_Rejected,
			ExpExecutorResult: Proposal_NotRun,
			WantDeliverLog:    "Proposal not accepted",
		},
		"Rejected without voters": {
			Src: tallySetup{
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 2,
			},
			ExpResult:         Proposal_Rejected,
			ExpExecutorResult: Proposal_NotRun,
			WantDeliverLog:    "Proposal not accepted",
		},
		"Rejected without enough votes: 2/3": {
			Src: tallySetup{
				yes:                   6,
				threshold:             Fraction{Numerator: 2, Denominator: 3},
				totalWeightElectorate: 9,
			},
			ExpResult:         Proposal_Rejected,
			ExpExecutorResult: Proposal_NotRun,
			WantDeliverLog:    "Proposal not accepted",
		},
		"Accepted with quorum and acceptance thresholds exceeded: 5/9": {
			Src: tallySetup{
				yes:                   5,
				quorum:                &Fraction{Numerator: 1, Denominator: 2},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult:         Proposal_Accepted,
			ExpExecutorResult: Proposal_Success,
			WantDeliverLog:    "Proposal accepted: execution success",
		},
		"Rejected with quorum thresholds not exceeded": {
			Src: tallySetup{
				yes:                   4,
				quorum:                &Fraction{Numerator: 1, Denominator: 2},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult:         Proposal_Rejected,
			ExpExecutorResult: Proposal_NotRun,
			WantDeliverLog:    "Proposal not accepted",
		},
		"Accepted with quorum and acceptance thresholds exceeded: 4/9": {
			Src: tallySetup{
				yes:                   4,
				no:                    1,
				quorum:                &Fraction{Numerator: 1, Denominator: 2},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult:         Proposal_Accepted,
			ExpExecutorResult: Proposal_Success,
			WantDeliverLog:    "Proposal accepted: execution success",
		},
		"Rejected with majority No": {
			Src: tallySetup{
				yes:                   4,
				no:                    5,
				quorum:                &Fraction{Numerator: 1, Denominator: 2},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult:         Proposal_Rejected,
			ExpExecutorResult: Proposal_NotRun,
			WantDeliverLog:    "Proposal not accepted",
		},
		"Rejected by single No when unanimity required": {
			Src: tallySetup{
				yes:                   8,
				no:                    1,
				quorum:                &Fraction{Numerator: 1, Denominator: 2},
				threshold:             Fraction{Numerator: 1, Denominator: 1},
				totalWeightElectorate: 9,
			},
			ExpResult:         Proposal_Rejected,
			ExpExecutorResult: Proposal_NotRun,
			WantDeliverLog:    "Proposal not accepted",
		},
		"Rejected by missing vote when all required": {
			Src: tallySetup{
				yes:                   8,
				quorum:                &Fraction{Numerator: 1, Denominator: 1},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult:         Proposal_Rejected,
			ExpExecutorResult: Proposal_NotRun,
			WantDeliverLog:    "Proposal not accepted",
		},
		"Accept on quorum fraction 1/1": {
			Src: tallySetup{
				yes:                   8,
				abstain:               1,
				quorum:                &Fraction{Numerator: 1, Denominator: 1},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult:         Proposal_Accepted,
			ExpExecutorResult: Proposal_Success,
			WantDeliverLog:    "Proposal accepted: execution success",
		},
		"Accepted with quorum and acceptance thresholds exceeded: 3/9": {
			Src: tallySetup{
				yes:                   3,
				abstain:               2,
				quorum:                &Fraction{Numerator: 1, Denominator: 2},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult:         Proposal_Accepted,
			ExpExecutorResult: Proposal_Success,
			WantDeliverLog:    "Proposal accepted: execution success",
		},
		"Accepted by single Yes and neutral abstains": {
			Src: tallySetup{
				yes:                   1,
				abstain:               4,
				quorum:                &Fraction{Numerator: 1, Denominator: 2},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult:         Proposal_Accepted,
			ExpExecutorResult: Proposal_Success,
			WantDeliverLog:    "Proposal accepted: execution success",
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
			ExpResult:         Proposal_Rejected,
			ExpExecutorResult: Proposal_NotRun,
			WantDeliverLog:    "Proposal not accepted",
		},
		"Accepted with acceptance thresholds < quorum": {
			Src: tallySetup{
				yes:                   2,
				abstain:               5,
				quorum:                &Fraction{Numerator: 2, Denominator: 3},
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 9,
			},
			ExpResult:         Proposal_Accepted,
			ExpExecutorResult: Proposal_Success,
			WantDeliverLog:    "Proposal accepted: execution success",
		},
		"Accepted with quorum and acceptance thresholds require all votes": {
			Src: tallySetup{
				yes:                   9,
				quorum:                &Fraction{Numerator: 1, Denominator: 1},
				threshold:             Fraction{Numerator: 1, Denominator: 1},
				totalWeightElectorate: 9,
			},
			ExpResult:         Proposal_Accepted,
			ExpExecutorResult: Proposal_Success,
			WantDeliverLog:    "Proposal accepted: execution success",
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
			ExpResult:         Proposal_Accepted,
			ExpExecutorResult: Proposal_Success,
			WantDeliverLog:    "Proposal accepted: execution success",
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
			ExpResult:         Proposal_Rejected,
			ExpExecutorResult: Proposal_NotRun,
			WantDeliverLog:    "Proposal not accepted",
		},
		"Updates latest electorate on success": {
			Init: func(t *testing.T, db weave.KVStore) {
				// update electorate for a new version
				bucket := NewElectorateBucket()
				_, obj, err := bucket.GetLatestVersion(db, weavetest.SequenceID(1))
				if err != nil {
					t.Fatalf("unexpected error: %+v", err)
				}
				e, _ := asElectorate(obj)
				e.Electors = []Elector{{hAlice, 1}, {hBobby, 10}, {hCharlie, 2}}
				e.TotalElectorateWeight = 13

				// to execute properly, the election rule that is tallied must be admin of the electorate in questions
				e.Admin = ElectionCondition(weavetest.SequenceID(1)).Address()

				if _, err := bucket.Update(db, weavetest.SequenceID(1), e); err != nil {
					t.Fatalf("unexpected error: %+v", err)
				}
			},
			Mods: func(ctx weave.Context, p *Proposal) {
				p.RawOption = genElectorateOptions(t, Elector{hAlice, 10})
				p.VotingEndTime = unixBlockTime(t, ctx) - 1
			},
			Src: tallySetup{
				yes:                   10,
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 11,
			},
			ExpResult:         Proposal_Accepted,
			ExpExecutorResult: Proposal_Success,
			PostChecks: func(t *testing.T, db weave.KVStore) {
				_, obj, err := NewElectorateBucket().GetLatestVersion(db, weavetest.SequenceID(1))
				if err != nil {
					t.Fatalf("unexpected error: %+v", err)
				}
				elect, _ := asElectorate(obj)
				if exp, got := uint32(3), elect.Version; exp != got {
					t.Errorf("expected %v but got %v", exp, got)
				}
				got := elect.Electors
				exp := []Elector{{hAlice, 10}, {hBobby, 10}, {hCharlie, 2}}
				sortByAddress(exp)
				if !reflect.DeepEqual(exp, got) {
					t.Errorf("expected %v but got %v", exp, got)
				}
				if exp, got := uint64(22), elect.TotalElectorateWeight; exp != got {
					t.Errorf("expected %v but got %v", exp, got)
				}
			},
			WantDeliverLog: "Proposal accepted: execution success",
		},
		"Completes tally even when executor fails": {
			Init: func(t *testing.T, db weave.KVStore) {
				// update electorate for a new version without Alice
				bucket := NewElectorateBucket()
				_, obj, err := bucket.GetLatestVersion(db, weavetest.SequenceID(1))
				if err != nil {
					t.Fatalf("unexpected error: %+v", err)
				}
				e, _ := asElectorate(obj)

				// this will fail as only elector is bobby, and we try to remove alice
				e.Electors = []Elector{{hBobby, 10}}
				e.TotalElectorateWeight = 10

				// to execute properly, the election rule that is tallied must be admin of the electorate in questions
				e.Admin = ElectionCondition(weavetest.SequenceID(1)).Address()

				if _, err := bucket.Update(db, weavetest.SequenceID(1), e); err != nil {
					t.Fatalf("unexpected error: %+v", err)
				}
			},
			Mods: func(ctx weave.Context, p *Proposal) {
				p.RawOption = genElectorateOptions(t, Elector{hAlice, 0})
				p.VotingEndTime = unixBlockTime(t, ctx) - 1
			},
			Src: tallySetup{
				yes:                   10,
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 11,
			},
			// Even if the execution failed, we update the tally state properly
			ExpResult:         Proposal_Accepted,
			ExpExecutorResult: Proposal_Failure,
			WantDeliverErr:    nil,
			WantDeliverLog:    "Proposal accepted: execution error:",
		},
		"Does not update an electorate when rejected": {
			Mods: func(ctx weave.Context, p *Proposal) {
				p.RawOption = genElectorateOptions(t, Elector{hAlice, 10})
				p.VotingEndTime = unixBlockTime(t, ctx) - 1
			},
			Src: tallySetup{
				yes:                   1,
				no:                    10,
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 11,
			},
			ExpResult:         Proposal_Rejected,
			ExpExecutorResult: Proposal_NotRun,
			PostChecks: func(t *testing.T, db weave.KVStore) {
				_, obj, err := NewElectorateBucket().GetLatestVersion(db, weavetest.SequenceID(1))
				if err != nil {
					t.Fatalf("unexpected error: %+v", err)
				}
				elect, _ := asElectorate(obj)
				got := elect.Electors
				exp := []Elector{{hAlice, 1}, {hBobby, 10}}
				sortByAddress(exp)
				if !reflect.DeepEqual(exp, got) {
					t.Errorf("expected %v but got %v", exp, got)
				}
				if exp, got := uint64(11), elect.TotalElectorateWeight; exp != got {
					t.Errorf("expected %v but got %v", exp, got)
				}
			},
			WantDeliverLog: "Proposal not accepted",
		},
		"Fails on second tally": {
			Mods: func(_ weave.Context, p *Proposal) {
				p.Status = Proposal_Closed
			},
			Src: tallySetup{
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 11,
			},
			WantDeliverErr:    errors.ErrState,
			ExpResult:         Proposal_Accepted,
			ExpExecutorResult: Proposal_Success,
			WantDeliverLog:    "Proposal not accepted",
		},
		"Fails on tally before end date": {
			Mods: func(ctx weave.Context, p *Proposal) {
				p.VotingEndTime = unixBlockTime(t, ctx) + 1
			},
			Src: tallySetup{
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 11,
			},
			WantDeliverErr: errors.ErrState,
			ExpResult:      Proposal_Undefined,
			WantDeliverLog: "Proposal not accepted",
		},
		"Fails on tally at end date": {
			Mods: func(ctx weave.Context, p *Proposal) {
				p.VotingEndTime = unixBlockTime(t, ctx)
			},
			Src: tallySetup{
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 11,
			},
			WantDeliverErr: errors.ErrState,
			ExpResult:      Proposal_Undefined,
			WantDeliverLog: "Proposal not accepted",
		},
		"Fails on withdrawn proposal": {
			Mods: func(ctx weave.Context, p *Proposal) {
				p.Status = Proposal_Withdrawn
			},
			Src: tallySetup{
				threshold:             Fraction{Numerator: 1, Denominator: 2},
				totalWeightElectorate: 11,
			},
			WantDeliverErr: errors.ErrState,
			ExpResult:      Proposal_Undefined,
			WantDeliverLog: "Proposal not accepted",
		},
	}
	rt := app.NewRouter()
	// Tally is registered for the cron, not for the usual routes.
	RegisterCronRoutes(rt, nil, decodeProposalOptions, proposalOptionsExecutor())

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, packageName)

			ctx := weave.WithBlockTime(context.Background(), time.Now().Round(time.Second))
			setupForTally := func(_ weave.Context, p *Proposal) {
				p.VoteState = NewTallyResult(spec.Src.quorum, spec.Src.threshold, spec.Src.totalWeightElectorate)
				p.VoteState.TotalYes = spec.Src.yes
				p.VoteState.TotalNo = spec.Src.no
				p.VoteState.TotalAbstain = spec.Src.abstain
				p.VotingEndTime = unixBlockTime(t, ctx) - 1
			}
			pBucket := withTextProposal(t, db, ctx, append([]ctxAwareMutator{setupForTally}, spec.Mods)...)
			if spec.Init != nil {
				spec.Init(t, db)
			}

			tx := &weavetest.Tx{
				Msg: &TallyMsg{
					Metadata:   &weave.Metadata{Schema: 1},
					ProposalID: weavetest.SequenceID(1),
				},
			}

			dres, err := rt.Deliver(ctx, db, tx)
			if !spec.WantDeliverErr.Is(err) {
				t.Fatalf("deliver expected: %+v  but got %+v", spec.WantDeliverErr, err)
			}
			if spec.WantDeliverErr != nil {
				return // skip further checks on expected error
			}
			if spec.WantDeliverLog != "" && !strings.HasPrefix(dres.Log, spec.WantDeliverLog) {
				t.Errorf("want Log: %s\ngot Log: %s", spec.WantDeliverLog, dres.Log)
			}

			// check persisted result
			p, err := pBucket.GetProposal(db, weavetest.SequenceID(1))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if exp, got := spec.ExpResult, p.Result; exp != got {
				t.Errorf("expected result %v but got %v: vote state: %#v", exp, got, p.VoteState)
			}
			if exp, got := spec.ExpExecutorResult, p.ExecutorResult; exp != got {
				t.Errorf("expected executor result %v but got %v: vote state: %#v", exp, got, p.VoteState)
			}
			if exp, got := Proposal_Closed, p.Status; exp != got {
				t.Errorf("expected %v but got %v", exp, got)
			}
			if spec.PostChecks != nil {
				spec.PostChecks(t, db)
			}
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
				DiffElectors: []Elector{{Address: hAlice, Weight: 22}},
			},
			SignedBy: hBobbyCond,
			ExpModel: &Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Admin:                 hBobby,
				Title:                 "fooo",
				Electors:              []Elector{{Address: hAlice, Weight: 22}, {Address: hBobby, Weight: 10}},
				TotalElectorateWeight: 32,
				Version:               2,
			},
		},
		"Update to remove address": {
			Msg: UpdateElectorateMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: electorateID,
				DiffElectors: []Elector{{Address: hAlice, Weight: 0}},
			},
			SignedBy: hBobbyCond,
			ExpModel: &Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Admin:                 hBobby,
				Title:                 "fooo",
				Electors:              []Elector{{Address: hBobby, Weight: 10}},
				TotalElectorateWeight: 10,
				Version:               2,
			},
		},
		"Update to add a new address": {
			Msg: UpdateElectorateMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: electorateID,
				DiffElectors: []Elector{{Address: hCharlie, Weight: 2}},
			},
			SignedBy: hBobbyCond,
			ExpModel: &Electorate{
				Metadata:              &weave.Metadata{Schema: 1},
				Admin:                 hBobby,
				Title:                 "fooo",
				Electors:              []Elector{{Address: hAlice, Weight: 1}, {Address: hBobby, Weight: 10}, {Address: hCharlie, Weight: 2}},
				TotalElectorateWeight: 13,
				Version:               2,
			},
		},
		"Update by non owner should fail": {
			Msg: UpdateElectorateMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: electorateID,
				DiffElectors: []Elector{{Address: hAlice, Weight: 22}},
			},
			SignedBy:       hAliceCond,
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
		},
		"Update with too many electors should fail": {
			Msg: UpdateElectorateMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: electorateID,
				DiffElectors: buildElectors(2001),
			},
			SignedBy:       hBobbyCond,
			WantDeliverErr: errors.ErrInput,
		},
		"Update without electors should fail": {
			Msg: UpdateElectorateMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: electorateID,
			},
			SignedBy:       hBobbyCond,
			WantCheckErr:   errors.ErrEmpty,
			WantDeliverErr: errors.ErrEmpty,
		},
		"Duplicate electors should fail": {
			Msg: UpdateElectorateMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: electorateID,
				DiffElectors: []Elector{{Address: hAlice, Weight: 1}, {Address: hAlice, Weight: 2}},
			},
			SignedBy:       hBobbyCond,
			WantCheckErr:   errors.ErrDuplicate,
			WantDeliverErr: errors.ErrDuplicate,
		},
		"Empty address in electors should fail": {
			Msg: UpdateElectorateMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: electorateID,
				DiffElectors: []Elector{{Address: weave.Address{}, Weight: 1}},
			},
			SignedBy:       hBobbyCond,
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
			RegisterRoutes(rt, auth, decodeProposalOptions, nil, &weavetest.Cron{})
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
			_, obj, err := bucket.GetLatestVersion(db, res.Data)
			if err != nil {
				t.Fatalf("unexpected error: %+v", err)
			}
			elect, _ := asElectorate(obj)
			sortByAddress(spec.ExpModel.Electors)
			if exp, got := spec.ExpModel, elect; !reflect.DeepEqual(exp, got) {
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
				Metadata:       &weave.Metadata{Schema: 1},
				ElectionRuleID: electionRulesID,
				VotingPeriod:   weave.AsUnixDuration(12 * time.Hour),
				Threshold:      Fraction{Numerator: 2, Denominator: 3},
			},
			SignedBy: hBobbyCond,
			ExpModel: &ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Version:      2,
				Admin:        hBobby,
				ElectorateID: weavetest.SequenceID(1),
				Title:        "barr",
				VotingPeriod: weave.AsUnixDuration(12 * time.Hour),
				Threshold:    Fraction{Numerator: 2, Denominator: 3},
				Address:      Condition(electionRulesID).Address(),
			},
		},
		"Update with max voting time": {
			Msg: UpdateElectionRuleMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				ElectionRuleID: electionRulesID,
				VotingPeriod:   weave.AsUnixDuration(24 * 7 * 4 * time.Hour),
				Threshold:      Fraction{Numerator: 2, Denominator: 3},
			},
			SignedBy: hBobbyCond,
			ExpModel: &ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Version:      2,
				Admin:        hBobby,
				ElectorateID: weavetest.SequenceID(1),
				Title:        "barr",
				VotingPeriod: weave.AsUnixDuration(24 * 7 * 4 * time.Hour),
				Threshold:    Fraction{Numerator: 2, Denominator: 3},
				Address:      Condition(electionRulesID).Address(),
			},
		},
		"Update by non owner should fail": {
			Msg: UpdateElectionRuleMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				ElectionRuleID: electionRulesID,
				VotingPeriod:   weave.AsUnixDuration(12 * time.Hour),
				Threshold:      Fraction{Numerator: 2, Denominator: 3},
			},
			SignedBy:       hAliceCond,
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
		},
		"Update can set a new quorum rule": {
			Msg: UpdateElectionRuleMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				ElectionRuleID: electionRulesID,
				VotingPeriod:   weave.AsUnixDuration(24 * 7 * 4 * time.Hour),
				Threshold:      Fraction{Numerator: 2, Denominator: 3},
				Quorum:         &Fraction{Numerator: 6, Denominator: 7},
			},
			SignedBy: hBobbyCond,
			ExpModel: &ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Version:      2,
				Admin:        hBobby,
				ElectorateID: weavetest.SequenceID(1),
				Title:        "barr",
				VotingPeriod: weave.AsUnixDuration(24 * 7 * 4 * time.Hour),
				Threshold:    Fraction{Numerator: 2, Denominator: 3},
				Quorum:       &Fraction{Numerator: 6, Denominator: 7},
				Address:      Condition(electionRulesID).Address(),
			},
		},
		"Update can unset quorum rule": {
			Msg: UpdateElectionRuleMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				ElectionRuleID: electionRulesID,
				VotingPeriod:   weave.AsUnixDuration(24 * 7 * 4 * time.Hour),
				Threshold:      Fraction{Numerator: 2, Denominator: 3},
				Quorum:         nil,
			},
			SignedBy: hBobbyCond,
			ExpModel: &ElectionRule{
				Metadata:     &weave.Metadata{Schema: 1},
				Version:      2,
				Admin:        hBobby,
				ElectorateID: weavetest.SequenceID(1),
				Title:        "barr",
				VotingPeriod: weave.AsUnixDuration(24 * 7 * 4 * time.Hour),
				Threshold:    Fraction{Numerator: 2, Denominator: 3},
				Quorum:       nil,
				Address:      Condition(electionRulesID).Address(),
			},
		},
		"Threshold must be valid": {
			Msg: UpdateElectionRuleMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				ElectionRuleID: electionRulesID,
				VotingPeriod:   weave.AsUnixDuration(12 * time.Hour),
				Threshold:      Fraction{Numerator: 3, Denominator: 2},
			},
			SignedBy:       hBobbyCond,
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"voting period hours must not be empty": {
			Msg: UpdateElectionRuleMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				ElectionRuleID: electionRulesID,
				VotingPeriod:   0,
				Threshold:      Fraction{Numerator: 1, Denominator: 2},
			},
			SignedBy:       hBobbyCond,
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"voting period hours must not exceed max": {
			Msg: UpdateElectionRuleMsg{
				Metadata:       &weave.Metadata{Schema: 1},
				ElectionRuleID: electionRulesID,
				VotingPeriod:   weave.AsUnixDuration((24*7*4 + 1) * time.Hour),
				Threshold:      Fraction{Numerator: 1, Denominator: 2},
			},
			SignedBy:       hBobbyCond,
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
			RegisterRoutes(rt, auth, decodeProposalOptions, nil, &weavetest.Cron{})
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
			_, obj, err := bucket.GetLatestVersion(db, res.Data)
			if err != nil {
				t.Fatalf("unexpected error: %+v", err)
			}
			e, _ := asElectionRule(obj)
			if exp, got := spec.ExpModel, e; !reflect.DeepEqual(exp, got) {
				t.Errorf("expected %v but got %v", exp, got)
			}
		})
	}
}

func unixBlockTime(t testing.TB, ctx context.Context) weave.UnixTime {
	now, err := weave.BlockTime(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return weave.AsUnixTime(now)
}
