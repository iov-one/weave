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
	NewElectionRulesBucket().Register("electionrules", qr)
	NewElectorateBucket().Register("electorates", qr)
	NewProposalBucket().Register("proposal", qr)
}

// RegisterRoutes registers handlers for governance message processing.
func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	propBucket := NewProposalBucket()
	elecBucket := NewElectorateBucket()
	r.Handle(pathVoteMsg, &VoteHandler{
		auth:       auth,
		propBucket: propBucket,
		elecBucket: elecBucket,
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
}

func (h VoteHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	if _, _, _, err := h.validate(ctx, db, tx); err != nil {
		return res, err
	}
	res.GasAllocated += voteCost
	return res, nil

}

func (h VoteHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	vote, proposal, elector, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}
	if err := proposal.Vote(vote.Selected, *elector); err != nil {
		return res, err
	}
	return res, h.propBucket.Update(db, vote.ProposalId, proposal)
}

func (h VoteHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*VoteMsg, *TextProposal, *Elector, error) {
	var msg VoteMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, nil, errors.Wrap(err, "load msg")
	}
	proposal, err := h.propBucket.GetTextProposal(db, msg.ProposalId)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to load proposal")
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

	if proposal.HasVoted(voter) {
		return nil, nil, nil, errors.Wrap(errors.ErrInvalidState, "already voted")
	}

	electorate, err := h.elecBucket.GetElectorate(db, proposal.ElectorateId)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to load electorate")
	}

	elector, ok := electorate.Elector(voter)
	if !ok {
		return nil, nil, nil, errors.Wrap(errors.ErrUnauthorized, "not in participants list")
	}
	return &msg, proposal, elector, nil
}

type TallyHandler struct {
	auth   x.Authenticator
	bucket *ProposalBucket
}

func (h TallyHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return res, err
	}
	res.GasAllocated += tallyCost
	return res, nil

}

func (h TallyHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, proposal, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}
	if err := proposal.Tally(); err != nil {
		return res, err
	}
	if err := h.bucket.Update(db, msg.ProposalId, proposal); err != nil {
		return res, err
	}
	res.Tags = append(res.Tags, []common.KVPair{
		{Key: []byte(tagProposerID), Value: msg.ProposalId},
		{Key: []byte(tagAction), Value: []byte("tally")},
	}...)
	return res, nil
}

func (h TallyHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*TallyMsg, *TextProposal, error) {
	var msg TallyMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	proposal, err := h.bucket.GetTextProposal(db, msg.ProposalId)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to load proposal")
	}
	if proposal.Status != TextProposal_Undefined {
		return nil, nil, errors.Wrap(errors.ErrInvalidState, "tally executed before")
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

func (h TextProposalHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	if _, _, _, err := h.validate(ctx, db, tx); err != nil {
		return res, err
	}
	res.GasAllocated += proposalCost
	return res, nil

}

func (h TextProposalHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, rules, electorate, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}
	blockTime, _ := weave.BlockTime(ctx)

	proposal := &TextProposal{
		Title:           msg.Title,
		Description:     msg.Description,
		ElectionRuleId:  msg.ElectionRuleId,
		ElectorateId:    msg.ElectorateId,
		VotingStartTime: msg.StartTime,
		VotingEndTime:   msg.StartTime.Add(time.Duration(rules.VotingPeriodHours) * time.Hour),
		SubmissionTime:  weave.AsUnixTime(blockTime),
		Author:          msg.Author,
		VoteResult: TallyResult{
			TotalWeightElectorate: electorate.TotalWeightElectorate,
			Threshold:             rules.Threshold,
		},
		Status: TextProposal_Undefined,
	}

	obj := h.propBucket.Build(db, proposal)
	if err := h.propBucket.Save(db, obj); err != nil {
		return res, errors.Wrap(err, "failed to persist proposal")
	}
	res.Tags = append(res.Tags, []common.KVPair{
		{Key: []byte(tagProposerID), Value: obj.Key()},
		{Key: []byte(tagProposer), Value: msg.Author},
		{Key: []byte(tagAction), Value: []byte("create")},
	}...)
	res.Data = obj.Key()
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
	elect, err := h.elecBucket.GetElectorate(db, msg.ElectorateId)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to load electorate")
	}
	rules, err := h.rulesBucket.GetElectionRule(db, msg.ElectionRuleId)
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
