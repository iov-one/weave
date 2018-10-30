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

// withMultisig is a private method, as only this module
// can add a multisig signer
func withApproval(ctx weave.Context, id []byte, approvalType string) weave.Context {
	val, _ := ctx.Value(contextKeyApprovals).([]weave.Condition)
	if val == nil {
		return context.WithValue(ctx, contextKeyApprovals, []weave.Condition{ApprovalCondition(id, approvalType)})
	}

	return context.WithValue(ctx, contextKeyApprovals, append(val, ApprovalCondition(id, approvalType)))
}

// MultiSigCondition returns condition for a contract ID
func ApprovalCondition(id []byte, approvalType string) weave.Condition {
	return weave.NewCondition("approvals", approvalType, id)
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
