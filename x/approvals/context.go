package approvals

import (
	"context"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
)

type contextKey int // local to the multisig module

const (
	contextKeyApprovals contextKey = iota
)

// withApproval is a private method, as only this module
// can add a approved signer
func withApproval(ctx weave.Context, id []byte) weave.Context {
	val, _ := ctx.Value(contextKeyApprovals).([]weave.Condition)
	if val == nil {
		return context.WithValue(ctx, contextKeyApprovals, []weave.Condition{ApprovalCondition(id, "usage")})
	}

	return context.WithValue(ctx, contextKeyApprovals, append(val, ApprovalCondition(id, "usage")))
}

// ApprovalCondition returns condition for a given action and signer
func ApprovalCondition(id []byte, approvalType string) weave.Condition {
	return weave.NewCondition("approval", approvalType, id)
}

// Authenticate gets/sets permissions on the given context key
type Authenticate struct {
}

var _ x.Authenticator = Authenticate{}

// GetConditions returns permissions previously set on this context
func (a Authenticate) GetConditions(ctx weave.Context) []weave.Condition {
	// (val, ok) form to return nil instead of panic if unset
	val, _ := ctx.Value(contextKeyApprovals).([]weave.Condition)
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

func HasApprovals(ctx weave.Context, auth x.Authenticator, allowed []weave.Condition, requestedAction string) (bool, []weave.Condition) {
	var approved []weave.Condition
	for _, a := range allowed {
		_, action, addr, _ := a.Parse()
		if action == requestedAction {
			if auth.HasAddress(ctx, ApprovalCondition(addr, "usage").Address()) {
				approved = append(approved, a)
			}
		}
	}
	return len(approved) > 0, approved
}
