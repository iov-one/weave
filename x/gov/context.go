package gov

import (
	"context"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
)

type contextKey int

const (
	// private type creates an interface key for Context that cannot be accessed by any other package
	contextKeyGov contextKey = iota
	contextKeyProposal
)

type proposalWrapper struct {
	proposal   *Proposal
	proposalID []byte
}

func withElectionSuccess(ctx weave.Context, ruleID []byte) weave.Context {
	val, _ := ctx.Value(contextKeyGov).([]weave.Condition)
	return context.WithValue(ctx, contextKeyGov, append(val, ElectionCondition(ruleID)))
}

// ElectionCondition returns condition for an election rule ID.
func ElectionCondition(ruleID []byte) weave.Condition {
	return weave.NewCondition("gov", "rule", ruleID)
}

// Authenticate gets/sets permissions on the given context key.
type Authenticate struct {
}

var _ x.Authenticator = Authenticate{}

// GetConditions returns permissions previously set on this context.
func (a Authenticate) GetConditions(ctx weave.Context) []weave.Condition {
	// (val, ok) form to return nil instead of panic if unset
	val, _ := ctx.Value(contextKeyGov).([]weave.Condition)
	return val
}

// HasAddress returns true iff this address is in GetConditions.
func (a Authenticate) HasAddress(ctx weave.Context, addr weave.Address) bool {
	for _, s := range a.GetConditions(ctx) {
		if addr.Equals(s.Address()) {
			return true
		}
	}
	return false
}

func withProposal(ctx weave.Context, proposal *Proposal, proposalID []byte) weave.Context {
	return context.WithValue(ctx, contextKeyProposal, proposalWrapper{proposal: proposal, proposalID: proposalID})
}

// CtxProposal reads the the proposal and it's id from the context
func CtxProposal(ctx weave.Context) (*Proposal, []byte) {
	val, _ := ctx.Value(contextKeyProposal).(proposalWrapper)
	return val.proposal, val.proposalID
}
