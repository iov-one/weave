package gov

import (
	"fmt"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

const (
	proposalCost           = 0
	deleteProposalCost     = 0
	voteCost               = 0
	tallyCost              = 0
	updateElectorateCost   = 0
	updateElectionRuleCost = 0
	textResolutionCost     = 0
)

const packageName = "gov"

// RegisterQuery registers governance buckets for querying.
func RegisterQuery(qr weave.QueryRouter) {
	NewElectionRulesBucket().Register("electionRules", qr)
	NewElectorateBucket().Register("electorates", qr)
	NewProposalBucket().Register("proposal", qr)
	NewVoteBucket().Register("vote", qr)
}

// RegisterRoutes registers handlers for governance message processing.
func RegisterRoutes(r weave.Registry, auth x.Authenticator, decoder OptionDecoder, executor Executor) {
	r = migration.SchemaMigratingRegistry(packageName, r)
	r.Handle(pathVoteMsg, newVoteHandler(auth))
	r.Handle(pathTallyMsg, newTallyHandler(auth, decoder, executor))
	r.Handle(pathCreateProposalMsg, newCreateProposalHandler(auth, decoder))
	r.Handle(pathDeleteProposalMsg, newDeleteProposalHandler(auth))
	r.Handle(pathUpdateElectorateMsg, newUpdateElectorateHandler(auth))
	r.Handle(pathUpdateElectionRulesMsg, newUpdateElectionRuleHandler(auth))
	// We do NOT register the TextResultionHandler here... this is only for the proposal Executor
}

// RegisterBasicProposalRouters register the routes we accept for executing governance decisions.
func RegisterBasicProposalRouters(r weave.Registry, auth x.Authenticator) {
	r = migration.SchemaMigratingRegistry(packageName, r)
	r.Handle(pathUpdateElectorateMsg, newUpdateElectorateHandler(auth))
	r.Handle(pathUpdateElectionRulesMsg, newUpdateElectionRuleHandler(auth))
	r.Handle(pathTextResolutionMsg, newTextResolutionHandler(auth))
}

type VoteHandler struct {
	auth       x.Authenticator
	elecBucket *ElectorateBucket
	propBucket *ProposalBucket
	voteBucket *VoteBucket
}

func newVoteHandler(auth x.Authenticator) *VoteHandler {
	return &VoteHandler{
		auth:       auth,
		elecBucket: NewElectorateBucket(),
		propBucket: NewProposalBucket(),
		voteBucket: NewVoteBucket(),
	}
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

	oldVote, err := h.voteBucket.GetVote(db, voteMsg.ProposalID, vote.Elector.Address)
	if !errors.ErrNotFound.Is(err) { // we only need to "UndoCount" if there was a previous vote
		if err != nil {
			return nil, errors.Wrap(err, "failed to load vote")
		}
		if err := proposal.Common.UndoCountVote(*oldVote); err != nil {
			return nil, err
		}
	}

	if err := proposal.Common.CountVote(*vote); err != nil {
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

func (h VoteHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*VoteMsg, *Proposal, *Vote, error) {
	var msg VoteMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, nil, errors.Wrap(err, "load msg")
	}
	proposal, err := h.propBucket.GetProposal(db, msg.ProposalID)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to load proposal")
	}
	common := proposal.Common
	if common == nil {
		return nil, nil, nil, errors.Wrap(errors.ErrState, "proposal doesn't have common values set")
	}

	if common.Status != ProposalCommon_Submitted {
		return nil, nil, nil, errors.Wrap(errors.ErrState, "not in voting period")
	}
	blockTime, ok := weave.BlockTime(ctx)
	if !ok {
		return nil, nil, nil, errors.Wrap(errors.ErrHuman, "block time not set")
	}
	if !blockTime.After(common.VotingStartTime.Time()) {
		return nil, nil, nil, errors.Wrap(errors.ErrState, "vote before proposal start time")
	}
	if !blockTime.Before(common.VotingEndTime.Time()) {
		return nil, nil, nil, errors.Wrapf(errors.ErrState, "vote after proposal end time: %s < %s", common.VotingEndTime, blockTime)
	}

	voter := msg.Voter
	if voter == nil {
		voter = x.MainSigner(ctx, h.auth).Address()
	}
	obj, err := h.elecBucket.GetVersion(db, common.ElectorateRef)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to load electorate")
	}
	elect, err := asElectorate(obj)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "electorate")
	}
	elector, ok := elect.Elector(voter)
	if !ok {
		return nil, nil, nil, errors.Wrap(errors.ErrUnauthorized, "not in participants list")
	}
	if !h.auth.HasAddress(ctx, voter) {
		return nil, nil, nil, errors.Wrap(errors.ErrUnauthorized, "voter must sign msg")
	}
	vote := &Vote{
		Metadata: &weave.Metadata{Schema: 1},
		Elector:  *elector,
		Voted:    msg.Selected,
	}
	if err := vote.Validate(); err != nil {
		return nil, nil, nil, err
	}
	return &msg, proposal, vote, nil
}

type TallyHandler struct {
	auth       x.Authenticator
	propBucket *ProposalBucket
	elecBucket *ElectorateBucket
	decoder    OptionDecoder
	executor   Executor
}

func newTallyHandler(auth x.Authenticator, decoder OptionDecoder, executor Executor) *TallyHandler {
	return &TallyHandler{
		auth:       auth,
		propBucket: NewProposalBucket(),
		elecBucket: NewElectorateBucket(),
		decoder:    decoder,
		executor:   executor,
	}
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
	common := proposal.Common
	if common == nil {
		return nil, errors.Wrap(errors.ErrState, "missing base proposal information")
	}

	if err := common.Tally(); err != nil {
		return nil, err
	}

	if err := h.propBucket.Update(db, msg.ProposalID, proposal); err != nil {
		return nil, err
	}

	if common.Result != ProposalCommon_Accepted {
		return &weave.DeliverResult{Log: "Proposal not accepted"}, nil
	}

	// we only execute the store options upon success
	// if this fails... we should still return no error, so the tally update works
	// we just return the info from the executor in logs (tags?)
	opts, err := h.decoder(proposal.RawOption)
	if err != nil {
		return &weave.DeliverResult{Log: "Proposal accepted: error: cannot parse raw options"}, nil
	}
	if err := opts.Validate(); err != nil {
		return &weave.DeliverResult{Log: "Proposal accepted: error: options invalid"}, nil
	}

	// we add the vote ctx here, to authenticate results in the executor
	// ensure that the gov.Authenticator is used in those Handlers
	voteCtx := withElectionSuccess(ctx, common.ElectionRuleRef.ID)
	cstore, ok := db.(weave.CacheableKVStore)
	if !ok {
		return &weave.DeliverResult{Log: "Proposal accepted: error: need cachable kvstore"}, nil
	}
	subDB := cstore.CacheWrap()

	res, err := h.executor(voteCtx, subDB, opts)
	if err != nil {
		subDB.Discard()
		log := fmt.Sprintf("Proposal accepted: execution error: %v", err)
		return &weave.DeliverResult{Log: log}, nil
	}
	if err := subDB.Write(); err != nil {
		log := fmt.Sprintf("Proposal accepted: commit error: %v", err)
		return &weave.DeliverResult{Log: log}, nil
	}

	res.Log = "Proposal accepted: execution success"
	return res, nil
}

func (h TallyHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*TallyMsg, *Proposal, error) {
	var msg TallyMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	proposal, err := h.propBucket.GetProposal(db, msg.ProposalID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to load proposal")
	}
	common := proposal.Common
	if common == nil {
		return nil, nil, errors.Wrap(errors.ErrState, "missing base proposal information")
	}
	if common.Status != ProposalCommon_Submitted {
		return nil, nil, errors.Wrapf(errors.ErrState, "unexpected status: %s", common.Status.String())
	}
	blockTime, ok := weave.BlockTime(ctx)
	if !ok {
		return nil, nil, errors.Wrap(errors.ErrHuman, "block time not set")
	}
	if !blockTime.After(common.VotingEndTime.Time()) {
		return nil, nil, errors.Wrapf(errors.ErrState, "tally before proposal end time: block time: %v < end time: %v", blockTime, common.VotingEndTime.Time())
	}
	return &msg, proposal, nil
}

type CreateProposalHandler struct {
	auth        x.Authenticator
	decoder     OptionDecoder
	elecBucket  *ElectorateBucket
	propBucket  *ProposalBucket
	rulesBucket *ElectionRulesBucket
}

func newCreateProposalHandler(auth x.Authenticator, decoder OptionDecoder) *CreateProposalHandler {
	return &CreateProposalHandler{
		auth:        auth,
		decoder:     decoder,
		elecBucket:  NewElectorateBucket(),
		propBucket:  NewProposalBucket(),
		rulesBucket: NewElectionRulesBucket(),
	}
}

func (h CreateProposalHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: proposalCost}, nil

}

func (h CreateProposalHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, rule, electorate, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	// abort when an update electorate proposal exists for the one used
	blockTime, _ := weave.BlockTime(ctx)

	base := msg.Base
	if base == nil {
		return nil, errors.Wrap(errors.ErrInput, "missing base create proposal info")
	}

	proposal := &Proposal{
		Metadata: &weave.Metadata{Schema: 1},
		Common: &ProposalCommon{
			Title:           base.Title,
			Description:     base.Description,
			ElectionRuleRef: orm.VersionedIDRef{ID: base.ElectionRuleID, Version: rule.Version},
			ElectorateRef:   orm.VersionedIDRef{ID: rule.ElectorateID, Version: electorate.Version},
			VotingStartTime: base.StartTime,
			VotingEndTime:   base.StartTime.Add(rule.VotingPeriod.Duration()),
			SubmissionTime:  weave.AsUnixTime(blockTime),
			Author:          base.Author,
			VoteState:       NewTallyResult(rule.Quorum, rule.Threshold, electorate.TotalElectorateWeight),
			Status:          ProposalCommon_Submitted,
			Result:          ProposalCommon_Undefined,
		},
		RawOption: msg.RawOption,
	}

	obj, err := h.propBucket.Create(db, proposal)
	if err != nil {
		return nil, errors.Wrap(err, "failed to persist proposal")
	}

	return &weave.DeliverResult{Data: obj.Key()}, nil
}

func (h CreateProposalHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*CreateProposalMsg, *ElectionRule, *Electorate, error) {
	var msg CreateProposalMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, nil, errors.Wrap(err, "load msg")
	}
	base := msg.Base
	if base == nil {
		return nil, nil, nil, errors.Wrap(errors.ErrInput, "missing base create proposal info")
	}

	blockTime, ok := weave.BlockTime(ctx)
	if !ok {
		return nil, nil, nil, errors.Wrap(errors.ErrHuman, "block time not set")
	}
	if !base.StartTime.Time().After(blockTime) {
		return nil, nil, nil, errors.Wrapf(errors.ErrInput, "start time must be in the future: %s < %s", base.StartTime, blockTime)
	}
	if blockTime.Add(maxFutureStart).Before(base.StartTime.Time()) {
		return nil, nil, nil, errors.Wrapf(errors.ErrInput, "start time cam not be more than %s h in the future", maxFutureStart)
	}

	_, rObj, err := h.rulesBucket.GetLatestVersion(db, base.ElectionRuleID)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to load election rule")
	}
	rule, err := asElectionRule(rObj)
	if err != nil {
		return nil, nil, nil, err
	}

	_, obj, err := h.elecBucket.GetLatestVersion(db, rule.ElectorateID)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to load electorate")
	}
	elect, err := asElectorate(obj)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "electorate")
	}

	author := base.Author
	if author != nil {
		if !h.auth.HasAddress(ctx, author) {
			return nil, nil, nil, errors.Wrap(errors.ErrUnauthorized, "author's signature required")
		}
	} else {
		author = x.MainSigner(ctx, h.auth).Address()
	}
	base.Author = author

	opts, err := h.decoder(msg.RawOption)
	if err != nil {
		return nil, nil, nil, errors.Wrap(errors.ErrInput, "cannot parse raw options")
	}
	if err := opts.Validate(); err != nil {
		return nil, nil, nil, errors.Wrap(err, "options invalid")
	}

	return &msg, rule, elect, nil
}

type DeleteProposalHandler struct {
	auth       x.Authenticator
	propBucket *ProposalBucket
}

func newDeleteProposalHandler(auth x.Authenticator) *DeleteProposalHandler {
	return &DeleteProposalHandler{
		auth:       auth,
		propBucket: NewProposalBucket(),
	}
}

func (h DeleteProposalHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*DeleteProposalMsg, *Proposal, error) {
	var msg DeleteProposalMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	blockTime, ok := weave.BlockTime(ctx)
	if !ok {
		return nil, nil, errors.Wrap(errors.ErrHuman, "block time not set")
	}
	prop, err := h.propBucket.GetProposal(db, msg.ProposalID)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to load a proposal with id %s", msg.ProposalID)
	}
	common := prop.Common
	if common == nil {
		return nil, nil, errors.Wrap(errors.ErrState, "proposal missing common data")
	}

	if common.Status == ProposalCommon_Withdrawn {
		return nil, nil, errors.Wrap(errors.ErrState, "this proposal is already withdrawn")
	}
	if common.VotingStartTime.Time().Before(blockTime) {
		return nil, nil, errors.Wrap(errors.ErrImmutable, "voting has already started")
	}
	if !h.auth.HasAddress(ctx, common.Author) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "only the author can delete a proposal")
	}
	return &msg, prop, nil
}

func (h DeleteProposalHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: deleteProposalCost}, nil
}

func (h DeleteProposalHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, prop, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	prop.Common.Status = ProposalCommon_Withdrawn

	if err := h.propBucket.Update(db, msg.ProposalID, prop); err != nil {
		return nil, errors.Wrap(err, "failed to persist proposal")
	}

	return &weave.DeliverResult{}, nil
}

type UpdateElectorateHandler struct {
	auth       x.Authenticator
	propBucket *ProposalBucket
	elecBucket *ElectorateBucket
}

func newUpdateElectorateHandler(auth x.Authenticator) *UpdateElectorateHandler {
	return &UpdateElectorateHandler{
		auth:       auth,
		propBucket: NewProposalBucket(),
		elecBucket: NewElectorateBucket(),
	}
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
	// all good, let's update
	merger := newMerger(elect.Electors)
	_ = merger.merge(msg.DiffElectors)
	elect.Electors, elect.TotalElectorateWeight = merger.serialize()

	if _, err := h.elecBucket.Update(db, msg.ElectorateID, elect); err != nil {
		return nil, errors.Wrap(err, "failed to store update")
	}
	return &weave.DeliverResult{Data: msg.ElectorateID}, nil
}

func (h UpdateElectorateHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*UpdateElectorateMsg, *Electorate, error) {
	var msg UpdateElectorateMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	// get latest electorate version
	_, obj, err := h.elecBucket.GetLatestVersion(db, msg.ElectorateID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to load electorate")
	}
	elect, err := asElectorate(obj)
	if err != nil {
		return nil, nil, errors.Wrap(err, "electorate")
	}

	if !h.auth.HasAddress(ctx, elect.Admin) {
		return nil, nil, errors.ErrUnauthorized
	}
	if err := newMerger(elect.Electors).merge(msg.DiffElectors); err != nil {
		return nil, nil, err
	}
	return &msg, elect, nil
}

type UpdateElectionRuleHandler struct {
	auth       x.Authenticator
	propBucket *ProposalBucket
	ruleBucket *ElectionRulesBucket
}

func newUpdateElectionRuleHandler(auth x.Authenticator) *UpdateElectionRuleHandler {
	return &UpdateElectionRuleHandler{
		auth:       auth,
		propBucket: NewProposalBucket(),
		ruleBucket: NewElectionRulesBucket(),
	}
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
	rule.VotingPeriod = msg.VotingPeriod
	if _, err := h.ruleBucket.Update(db, msg.ElectionRuleID, rule); err != nil {
		return nil, errors.Wrap(err, "failed to store update")
	}
	return &weave.DeliverResult{Data: msg.ElectionRuleID}, nil
}

func (h UpdateElectionRuleHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*UpdateElectionRuleMsg, *ElectionRule, error) {
	var msg UpdateElectionRuleMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	_, obj, err := h.ruleBucket.GetLatestVersion(db, msg.ElectionRuleID)
	if err != nil {
		return nil, nil, err
	}
	rule, err := asElectionRule(obj)
	if err != nil {
		return nil, nil, err
	}
	if !h.auth.HasAddress(ctx, rule.Admin) {
		return nil, nil, errors.ErrUnauthorized
	}
	return &msg, rule, nil
}

type TextResolutionHandler struct {
	auth x.Authenticator
}

func newTextResolutionHandler(auth x.Authenticator) *TextResolutionHandler {
	// TODO: actually add a bucket to store resolutions
	return &TextResolutionHandler{
		auth: auth,
	}
}

func (h TextResolutionHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: textResolutionCost}, nil
}

func (h TextResolutionHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	_, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	// TODO: store this resolution somewhere
	return &weave.DeliverResult{}, nil
}

func (h TextResolutionHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*TextResolutionMsg, error) {
	var msg TextResolutionMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}
	// TODO: some auth?
	// if !h.auth.HasAddress(ctx, rule.Admin) {
	// 	return nil, errors.ErrUnauthorized
	// }
	return &msg, nil
}
