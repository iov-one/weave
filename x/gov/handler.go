package gov

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x"
)

const (
	newTextProposalCost = 0
	voteCost            = 0
	tallyCost           = 0
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
	// todo: set tag to mark voting ???
	return res, h.propBucket.Update(db, vote.ProposalId, proposal)
}

func (h VoteHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*VoteMsg, *TextProposal, *Elector, error) {
	var msg VoteMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, nil, errors.Wrap(err, "load msg")
	}
	proposal, err := h.propBucket.GetTextProposal(db, msg.ProposalId)
	if err != nil {
		return nil, nil, nil, err
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

	// todo: should we move the votes into it's own bucket to make use of DB query features?
	// would be the same for electors in electorate
	if proposal.HasVoted(voter) {
		return nil, nil, nil, errors.Wrap(errors.ErrInvalidState, "already voted")
	}

	electorate, err := h.elecBucket.GetElectorate(db, proposal.ElectorateId)
	if err != nil {
		return nil, nil, nil, err
	}
	// todo: would a decorator make sense for auth?
	elector, ok := electorate.Elector(voter)
	if !ok {
		return nil, nil, nil, errors.Wrap(errors.ErrUnauthorized, "not in participants list")
	}
	return &msg, proposal, elector, nil
}
