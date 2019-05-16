package gov

import (
	"context"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
)

type contextKey int // local to the gov module

const (
	contextKeyGov contextKey = iota
)

// withElectionSuccess is a private method, as only this module
// can add a gov signer
func withElectionSuccess(ctx weave.Context, ruleID []byte) weave.Context {
	val, _ := ctx.Value(contextKeyGov).([]weave.Condition)
	if val == nil {
		return context.WithValue(ctx, contextKeyGov, []weave.Condition{ElectionCondition(ruleID)})
	}

	return context.WithValue(ctx, contextKeyGov, append(val, ElectionCondition(ruleID)))
}

// ElectionCondition returns condition for an election rule ID
func ElectionCondition(ruleID []byte) weave.Condition {
	return weave.NewCondition("gov", "rule", ruleID)
}

// Authenticate gets/sets permissions on the given context key
type Authenticate struct {
}

var _ x.Authenticator = Authenticate{}

// GetConditions returns permissions previously set on this context
func (a Authenticate) GetConditions(ctx weave.Context) []weave.Condition {
	// (val, ok) form to return nil instead of panic if unset
	val, _ := ctx.Value(contextKeyGov).([]weave.Condition)
	if val == nil {
		return nil
	}
	return val
}

// HasAddress returns true iff this address is in GetConditions
func (a Authenticate) HasAddress(ctx weave.Context, addr weave.Address) bool {
	for _, s := range a.GetConditions(ctx) {
		if addr.Equals(s.Address()) {
			return true
		}
	}
	return false
}
