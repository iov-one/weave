package gov

import (
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
	"github.com/tendermint/tendermint/libs/common"
)

const (
	proposalCost           = 0
	deleteProposalCost     = 0
	voteCost               = 0
	tallyCost              = 0
	updateElectorateCost   = 0
	updateElectionRuleCost = 0
)

const (
	tagProposalID = "proposal-id"
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
	r.Handle(pathDeleteTextProposalMsg, &DeleteTextProposalHandler{
		auth:       auth,
		propBucket: propBucket,
	})
	r.Handle(pathUpdateElectorateMsg, &UpdateElectorateHandler{
		auth:       auth,
		propBucket: propBucket,
		elecBucket: elecBucket,
	})
	r.Handle(pathUpdateElectionRulesMsg, &UpdateElectionRuleHandler{
		auth:       auth,
		propBucket: propBucket,
		ruleBucket: NewElectionRulesBucket(),
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
			{Key: []byte(tagProposalID), Value: msg.ProposalID},
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
			{Key: []byte(tagProposalID), Value: obj.Key()},
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
	if blockTime.Add(maxFutureStartTimeHours).Before(msg.StartTime.Time()) {
		return nil, nil, nil, errors.Wrapf(errors.ErrInvalidInput, "start time cam not be more than %d h in the future", maxFutureStartTimeHours)
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

type DeleteTextProposalHandler struct {
	auth       x.Authenticator
	propBucket *ProposalBucket
}

func (h DeleteTextProposalHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*DeleteTextProposalMsg, *TextProposal, error) {
	var msg DeleteTextProposalMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	blockTime, ok := weave.BlockTime(ctx)
	if !ok {
		return nil, nil, errors.Wrap(errors.ErrHuman, "block time not set")
	}
	prop, err := h.propBucket.GetTextProposal(db, msg.ID)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to load a proposal with id %s", msg.ID)
	}
	if prop.Status == TextProposal_Withdrawn {
		return nil, nil, errors.Wrap(errors.ErrInvalidState, "this proposal is already withdrawn")
	}
	if prop.VotingStartTime.Time().After(blockTime) {
		return nil, nil, errors.Wrap(errors.ErrCannotBeModified, "voting has already started")
	}
	if h.auth.HasAddress(ctx, prop.Author) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "only the author can delete a proposal")
	}
	return &msg, prop, nil
}

func (h DeleteTextProposalHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: deleteProposalCost}, nil
}

func (h DeleteTextProposalHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, prop, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	prop.Status = TextProposal_Withdrawn

	if err := h.propBucket.Update(db, msg.ID, prop); err != nil {
		return nil, errors.Wrap(err, "failed to persist proposal")
	}

	res := &weave.DeliverResult{
		Data: msg.ID,
		Tags: []common.KVPair{
			{Key: []byte(tagProposalID), Value: msg.ID},
			{Key: []byte(tagProposer), Value: prop.Author},
			{Key: []byte(tagAction), Value: []byte("delete")},
		},
	}
	return res, nil
}

type UpdateElectorateHandler struct {
	auth       x.Authenticator
	propBucket *ProposalBucket
	elecBucket *ElectorateBucket
}

func (h UpdateElectorateHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, _, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: updateElectorateCost}, nil
}

func (h UpdateElectorateHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, elect, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	props, err := h.propBucket.GetByElectorate(db, msg.ElectorateID)
	if err != nil {
		return nil, err
	}
	for _, v := range props {
		if v.Status == TextProposal_Submitted {
			return nil, errors.Wrapf(errors.ErrInvalidState, "open proposal using this electorate: %q", v.Title)
		}
	}
	// all good, let's update
	elect.Electors = msg.Electors
	elect.TotalWeightElectorate = 0
	for _, v := range msg.Electors {
		elect.TotalWeightElectorate += uint64(v.Weight)
	}
	if err := h.elecBucket.Save(db, orm.NewSimpleObj(msg.ElectorateID, elect)); err != nil {
		return nil, errors.Wrap(err, "failed to store update")
	}
	return &weave.DeliverResult{Data: msg.ElectorateID}, nil
}

func (h UpdateElectorateHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*UpdateElectorateMsg, *Electorate, error) {
	var msg UpdateElectorateMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	e, err := h.elecBucket.GetElectorate(db, msg.ElectorateID)
	if err != nil {
		return nil, nil, err
	}
	if !h.auth.HasAddress(ctx, e.Admin) {
		return nil, nil, errors.ErrUnauthorized
	}
	return &msg, e, nil
}

type UpdateElectionRuleHandler struct {
	auth       x.Authenticator
	propBucket *ProposalBucket
	ruleBucket *ElectionRulesBucket
}

func (h UpdateElectionRuleHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, _, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: updateElectionRuleCost}, nil
}

func (h UpdateElectionRuleHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, rule, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	rule.Threshold = msg.Threshold
	rule.VotingPeriodHours = msg.VotingPeriodHours
	if err := h.ruleBucket.Save(db, orm.NewSimpleObj(msg.ElectionRuleID, rule)); err != nil {
		return nil, errors.Wrap(err, "failed to store update")
	}
	return &weave.DeliverResult{Data: msg.ElectionRuleID}, nil
}

func (h UpdateElectionRuleHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*UpdateElectionRuleMsg, *ElectionRule, error) {
	var msg UpdateElectionRuleMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	e, err := h.ruleBucket.GetElectionRule(db, msg.ElectionRuleID)
	if err != nil {
		return nil, nil, err
	}
	if !h.auth.HasAddress(ctx, e.Admin) {
		return nil, nil, errors.ErrUnauthorized
	}
	return &msg, e, nil
}
