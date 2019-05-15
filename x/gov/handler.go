package gov

import (
	"time"

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
func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	propBucket := NewProposalBucket()
	elecBucket := NewElectorateBucket()
	r.Handle(pathVoteMsg, migration.SchemaMigratingHandler(packageName, &VoteHandler{
		auth:       auth,
		propBucket: propBucket,
		elecBucket: elecBucket,
		voteBucket: NewVoteBucket(),
	}))
	r.Handle(pathTallyMsg, migration.SchemaMigratingHandler(packageName,
		NewTallyHandler(auth, propBucket, elecBucket)))
	r.Handle(pathCreateProposalMsg, migration.SchemaMigratingHandler(packageName, &CreateProposalHandler{
		auth:        auth,
		propBucket:  propBucket,
		elecBucket:  elecBucket,
		rulesBucket: NewElectionRulesBucket(),
	}))
	r.Handle(pathDeleteProposalMsg, migration.SchemaMigratingHandler(packageName, &DeleteProposalHandler{
		auth:       auth,
		propBucket: propBucket,
	}))
	r.Handle(pathUpdateElectorateMsg, migration.SchemaMigratingHandler(packageName, &UpdateElectorateHandler{
		auth:       auth,
		propBucket: propBucket,
		elecBucket: elecBucket,
	}))
	r.Handle(pathUpdateElectionRulesMsg, migration.SchemaMigratingHandler(packageName, &UpdateElectionRuleHandler{
		auth:       auth,
		propBucket: propBucket,
		ruleBucket: NewElectionRulesBucket(),
	}))
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
		return nil, nil, nil, errors.Wrap(errors.ErrState, "vote after proposal end time")
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
	loader     OptionLoader
	executor   Executor
}

func NewTallyHandler(auth x.Authenticator, bucket *ProposalBucket, electBucket *ElectorateBucket) *TallyHandler {
	return &TallyHandler{
		auth:       auth,
		propBucket: bucket,
		elecBucket: electBucket,
		// TODO: loader and executor
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
	if err := proposal.Tally(); err != nil {
		return nil, err
	}
	common := proposal.Common
	if common == nil {
		return nil, errors.Wrap(errors.ErrState, "missing base proposal information")
	}

	if common.Result == ProposalCommon_Accepted {
		// TODO: rewrite this logic based on demo
		// exec, ok := h.typeHandler[proposal.Type]
		// if !ok {
		// 	return nil, errors.Wrapf(errors.ErrState, "unsupported type: %s", proposal.Type)
		// }
		// if err := exec(db, proposal); err != nil {
		// 	return nil, errors.Wrapf(err, "execution failed for type: %v", proposal.Type)
		// }
	}
	return &weave.DeliverResult{}, h.propBucket.Update(db, msg.ProposalID, proposal)
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
		return nil, nil, errors.Wrap(errors.ErrState, "tally before proposal end time")
	}
	return &msg, proposal, nil
}

// TODO: this needs to be redone
// // updateElectorateExecutor contains logic to update an electorate.
// func updateElectorateExecutor(elecBucket *ElectorateBucket) TallyResultExecutor {
// 	return func(db weave.KVStore, proposal *Proposal) error {
// 		common := proposal.Common
// 		if common == nil {
// 			return errors.Wrap(errors.ErrState, "missing base proposal information")
// 		}

// 		details := common.GetElectorateUpdateDetails()
// 		if details == nil {
// 			return errors.Wrap(errors.ErrHuman, "details not set")
// 		}
// 		// merge diff
// 		_, obj, err := elecBucket.GetLatestVersion(db, proposal.ElectorateRef.ID)
// 		if err != nil {
// 			return errors.Wrap(err, "failed to load electorate")
// 		}
// 		elect, err := asElectorate(obj)
// 		if err != nil {
// 			return err
// 		}
// 		merger := newMerger(elect.Electors)
// 		if err := merger.merge(details.DiffElectors); err != nil {
// 			return errors.Wrap(err, "merge failed")
// 		}
// 		elect.Electors, elect.TotalElectorateWeight = merger.serialize()
// 		if _, err := elecBucket.Update(db, proposal.ElectorateRef.ID, elect); err != nil {
// 			return errors.Wrap(err, "failed to store electorate update")
// 		}
// 		return nil
// 	}
// }

type CreateProposalHandler struct {
	auth        x.Authenticator
	elecBucket  *ElectorateBucket
	propBucket  *ProposalBucket
	rulesBucket *ElectionRulesBucket
	loader      OptionLoader
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
			ElectorateRef:   orm.VersionedIDRef{ID: base.ElectorateID, Version: electorate.Version},
			VotingStartTime: base.StartTime,
			VotingEndTime:   base.StartTime.Add(time.Duration(rule.VotingPeriodHours) * time.Hour),
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
		return nil, nil, nil, errors.Wrap(errors.ErrInput, "start time must be in the future")
	}
	if blockTime.Add(maxFutureStartTimeHours).Before(base.StartTime.Time()) {
		return nil, nil, nil, errors.Wrapf(errors.ErrInput, "start time cam not be more than %d h in the future", maxFutureStartTimeHours)
	}
	// get latest electorate version
	_, obj, err := h.elecBucket.GetLatestVersion(db, base.ElectorateID)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to load electorate")
	}
	elect, err := asElectorate(obj)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "electorate")
	}

	_, rObj, err := h.rulesBucket.GetLatestVersion(db, base.ElectionRuleID)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to load election rule")
	}
	rule, err := asElectionRule(rObj)
	if err != nil {
		return nil, nil, nil, err
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

	// TODO: validate RawOptions using OptionsLoader
	return &msg, rule, elect, nil
}

// type ElectorateUpdateProposalHandler struct {
// 	auth        x.Authenticator
// 	elecBucket  *ElectorateBucket
// 	propBucket  *ProposalBucket
// 	rulesBucket *ElectionRulesBucket
// }

// func (h ElectorateUpdateProposalHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
// 	if _, _, _, err := h.validate(ctx, db, tx); err != nil {
// 		return nil, err
// 	}
// 	return &weave.CheckResult{GasAllocated: proposalCost}, nil

// }

// func (h ElectorateUpdateProposalHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
// 	msg, rules, electorate, err := h.validate(ctx, db, tx)
// 	if err != nil {
// 		return nil, err
// 	}
// 	blockTime, _ := weave.BlockTime(ctx)

// 	proposal := &Proposal{
// 		Metadata:        &weave.Metadata{Schema: 1},
// 		Type:            Proposal_UpdateElectorate,
// 		Title:           msg.Title,
// 		Description:     msg.Description,
// 		ElectionRuleRef: electorate.UpdateElectionRuleRef,
// 		ElectorateRef:   orm.VersionedIDRef{ID: msg.ElectorateID, Version: electorate.Version},
// 		VotingStartTime: msg.StartTime,
// 		VotingEndTime:   msg.StartTime.Add(time.Duration(rules.VotingPeriodHours) * time.Hour),
// 		SubmissionTime:  weave.AsUnixTime(blockTime),
// 		Author:          msg.Author,
// 		VoteState:       NewTallyResult(rules.Quorum, rules.Threshold, electorate.TotalElectorateWeight),
// 		Status:          Proposal_Submitted,
// 		Result:          Proposal_Undefined,
// 		Details: &Proposal_ElectorateUpdateDetails{&ElectorateUpdatePayload{
// 			msg.DiffElectors,
// 		}},
// 	}

// 	obj, err := h.propBucket.Create(db, proposal)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "failed to persist proposal")
// 	}

// 	return &weave.DeliverResult{Data: obj.Key()}, nil
// }

// func (h ElectorateUpdateProposalHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*CreateElectorateUpdateProposalMsg, *ElectionRule, *Electorate, error) {
// 	var msg CreateElectorateUpdateProposalMsg
// 	if err := weave.LoadMsg(tx, &msg); err != nil {
// 		return nil, nil, nil, errors.Wrap(err, "load msg")
// 	}
// 	blockTime, ok := weave.BlockTime(ctx)
// 	if !ok {
// 		return nil, nil, nil, errors.Wrap(errors.ErrHuman, "block time not set")
// 	}
// 	if !msg.StartTime.Time().After(blockTime) {
// 		return nil, nil, nil, errors.Wrap(errors.ErrInput, "start time must be in the future")
// 	}
// 	if blockTime.Add(maxFutureStartTimeHours).Before(msg.StartTime.Time()) {
// 		return nil, nil, nil, errors.Wrapf(errors.ErrInput, "start time cam not be more than %d h in the future", maxFutureStartTimeHours)
// 	}
// 	// get latest electorate version
// 	_, obj, err := h.elecBucket.GetLatestVersion(db, msg.ElectorateID)
// 	if err != nil {
// 		return nil, nil, nil, errors.Wrap(err, "failed to load electorate")
// 	}
// 	elect, err := asElectorate(obj)
// 	if err != nil {
// 		return nil, nil, nil, errors.Wrap(err, "electorate")
// 	}
// 	if err := newMerger(elect.Electors).merge(msg.DiffElectors); err != nil {
// 		return nil, nil, nil, err
// 	}
// 	// get associated rule
// 	rObj, err := h.rulesBucket.GetVersion(db, elect.UpdateElectionRuleRef)
// 	if err != nil {
// 		return nil, nil, nil, errors.Wrap(err, "failed to load election rule")
// 	}
// 	rule, err := asElectionRule(rObj)
// 	if err != nil {
// 		return nil, nil, nil, err
// 	}

// 	author := msg.Author
// 	if author != nil {
// 		if !h.auth.HasAddress(ctx, author) {
// 			return nil, nil, nil, errors.Wrap(errors.ErrUnauthorized, "author's signature required")
// 		}
// 	} else {
// 		author = x.MainSigner(ctx, h.auth).Address()
// 	}
// 	msg.Author = author
// 	return &msg, rule, elect, nil
// }

type DeleteProposalHandler struct {
	auth       x.Authenticator
	propBucket *ProposalBucket
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
	prop, err := h.propBucket.GetProposal(db, msg.ID)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to load a proposal with id %s", msg.ID)
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

	if err := h.propBucket.Update(db, msg.ID, prop); err != nil {
		return nil, errors.Wrap(err, "failed to persist proposal")
	}

	return &weave.DeliverResult{}, nil
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
