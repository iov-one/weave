package gov

import (
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x"
	"github.com/tendermint/tendermint/libs/common"
)

const (
	proposalCost = 0
	voteCost     = 0
	tallyCost    = 0
)

const (
	tagProposerID = "proposal-id"
	tagAction     = "action"
	tagProposer   = "proposer"
)

// RegisterQuery registers governance buckets for querying.
func RegisterQuery(qr weave.QueryRouter) {
	NewElectionRulesBucket().Register("electionRules", qr)
	NewElectorateBucket().Register("electorates", qr)
	NewProposalBucket().Register("proposal", qr)
	NewVoteBucket().Register("vote", qr)
}

// RegisterRoutes registers handlers for governance message processing.
func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	propBucket := NewProposalBucket()
	elecBucket := NewElectorateBucket()
	r.Handle(pathVoteMsg, &VoteHandler{
		auth:       auth,
		propBucket: propBucket,
		elecBucket: elecBucket,
		voteBucket: NewVoteBucket(),
	})
	r.Handle(pathTallyMsg, &TallyHandler{
		auth:   auth,
		bucket: propBucket,
	})
	r.Handle(pathCreateTextProposalMsg, &TextProposalHandler{
		auth:        auth,
		propBucket:  propBucket,
		elecBucket:  elecBucket,
		rulesBucket: NewElectionRulesBucket(),
	})
}

type VoteHandler struct {
	auth       x.Authenticator
	elecBucket *ElectorateBucket
	propBucket *ProposalBucket
	voteBucket *VoteBucket
}

func (h VoteHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: voteCost}, nil

}

func (h VoteHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	voteMsg, proposal, vote, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	switch oldVote, err := h.voteBucket.GetVote(db, voteMsg.ProposalID, vote.Elector.Address); {
	case errors.ErrNotFound.Is(err): // not voted before: skip undo
	case err != nil:
		return nil, errors.Wrap(err, "failed to load vote")
	default:
		if err := proposal.UndoCountVote(*oldVote); err != nil {
			return nil, err
		}
	}

	if err := proposal.CountVote(*vote); err != nil {
		return nil, err
	}
	if err = h.voteBucket.Save(db, h.voteBucket.Build(db, voteMsg.ProposalID, *vote)); err != nil {
		return nil, errors.Wrap(err, "failed to store vote")
	}
	if err := h.propBucket.Update(db, voteMsg.ProposalID, proposal); err != nil {
		return nil, err
	}
	return &weave.DeliverResult{}, nil
}

func (h VoteHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*VoteMsg, *TextProposal, *Vote, error) {
	var msg VoteMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, nil, errors.Wrap(err, "load msg")
	}
	proposal, err := h.propBucket.GetTextProposal(db, msg.ProposalID)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to load proposal")
	}
	if proposal.Status != TextProposal_Submitted {
		return nil, nil, nil, errors.Wrap(errors.ErrInvalidState, "not in voting period")
	}
	blockTime, ok := weave.BlockTime(ctx)
	if !ok {
		return nil, nil, nil, errors.Wrap(errors.ErrHuman, "block time not set")
	}
	if !blockTime.After(proposal.VotingStartTime.Time()) {
		return nil, nil, nil, errors.Wrap(errors.ErrInvalidState, "vote before proposal start time")
	}
	if !blockTime.Before(proposal.VotingEndTime.Time()) {
		return nil, nil, nil, errors.Wrap(errors.ErrInvalidState, "vote after proposal end time")
	}

	voter := msg.Voter
	if voter == nil {
		voter = x.MainSigner(ctx, h.auth).Address()
	}
	electorate, err := h.elecBucket.GetElectorate(db, proposal.ElectorateID)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to load electorate")
	}

	elector, ok := electorate.Elector(voter)
	if !ok {
		return nil, nil, nil, errors.Wrap(errors.ErrUnauthorized, "not in participants list")
	}
	if !h.auth.HasAddress(ctx, voter) {
		return nil, nil, nil, errors.Wrap(errors.ErrUnauthorized, "voter must sign msg")
	}
	vote := &Vote{Elector: *elector, Voted: msg.Selected}
	if err := vote.Validate(); err != nil {
		return nil, nil, nil, err
	}
	return &msg, proposal, vote, nil
}

type TallyHandler struct {
	auth   x.Authenticator
	bucket *ProposalBucket
}

func (h TallyHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: tallyCost}, nil

}

func (h TallyHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, proposal, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	if err := proposal.Tally(); err != nil {
		return nil, err
	}
	if err := h.bucket.Update(db, msg.ProposalID, proposal); err != nil {
		return nil, err
	}

	res := &weave.DeliverResult{
		Tags: []common.KVPair{
			{Key: []byte(tagProposerID), Value: msg.ProposalID},
			{Key: []byte(tagAction), Value: []byte("tally")},
		},
	}
	return res, nil
}

func (h TallyHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*TallyMsg, *TextProposal, error) {
	var msg TallyMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	proposal, err := h.bucket.GetTextProposal(db, msg.ProposalID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to load proposal")
	}
	if proposal.Status != TextProposal_Submitted {
		return nil, nil, errors.Wrapf(errors.ErrInvalidState, "unexpected status: %s", proposal.Status.String())
	}
	blockTime, ok := weave.BlockTime(ctx)
	if !ok {
		return nil, nil, errors.Wrap(errors.ErrHuman, "block time not set")
	}
	if !blockTime.After(proposal.VotingEndTime.Time()) {
		return nil, nil, errors.Wrap(errors.ErrInvalidState, "tally before proposal end time")
	}
	return &msg, proposal, nil
}

type TextProposalHandler struct {
	auth        x.Authenticator
	elecBucket  *ElectorateBucket
	propBucket  *ProposalBucket
	rulesBucket *ElectionRulesBucket
}

func (h TextProposalHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: proposalCost}, nil

}

func (h TextProposalHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, rules, electorate, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	blockTime, _ := weave.BlockTime(ctx)

	proposal := &TextProposal{
		Title:           msg.Title,
		Description:     msg.Description,
		ElectionRuleID:  msg.ElectionRuleID,
		ElectorateID:    msg.ElectorateID,
		VotingStartTime: msg.StartTime,
		VotingEndTime:   msg.StartTime.Add(time.Duration(rules.VotingPeriodHours) * time.Hour),
		SubmissionTime:  weave.AsUnixTime(blockTime),
		Author:          msg.Author,
		VoteState: TallyResult{
			TotalWeightElectorate: electorate.TotalWeightElectorate,
			Threshold:             rules.Threshold,
		},
		Status: TextProposal_Submitted,
		Result: TextProposal_Undefined,
	}

	obj, err := h.propBucket.Build(db, proposal)
	if err != nil {
		return nil, err
	}
	if err := h.propBucket.Save(db, obj); err != nil {
		return nil, errors.Wrap(err, "failed to persist proposal")
	}

	res := &weave.DeliverResult{
		Data: obj.Key(),
		Tags: []common.KVPair{
			{Key: []byte(tagProposerID), Value: obj.Key()},
			{Key: []byte(tagProposer), Value: msg.Author},
			{Key: []byte(tagAction), Value: []byte("create")},
		},
	}
	return res, nil
}

func (h TextProposalHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*CreateTextProposalMsg, *ElectionRule, *Electorate, error) {
	var msg CreateTextProposalMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, nil, errors.Wrap(err, "load msg")
	}
	blockTime, ok := weave.BlockTime(ctx)
	if !ok {
		return nil, nil, nil, errors.Wrap(errors.ErrHuman, "block time not set")
	}
	if !msg.StartTime.Time().After(blockTime) {
		return nil, nil, nil, errors.Wrap(errors.ErrInvalidInput, "start time must be in the future")
	}
	elect, err := h.elecBucket.GetElectorate(db, msg.ElectorateID)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to load electorate")
	}
	rules, err := h.rulesBucket.GetElectionRule(db, msg.ElectionRuleID)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to load election rules")
	}
	author := msg.Author
	if author != nil {
		if !h.auth.HasAddress(ctx, author) {
			return nil, nil, nil, errors.Wrap(errors.ErrUnauthorized, "author's signature required")
		}
	} else {
		author = x.MainSigner(ctx, h.auth).Address()
	}
	msg.Author = author
	return &msg, rules, elect, nil
}
