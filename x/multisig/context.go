package multisig

import (
	"context"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
)

type contextKey int // local to the multisig module

const (
	contextKeyMultisig contextKey = iota
)

// withMultisig is a private method, as only this module
// can add a multisig signer
func withMultisig(ctx context.Context, info weave.BlockInfo, id []byte) context.Context {
	val, _ := ctx.Value(contextKeyMultisig).([]weave.Condition)
	if val == nil {
		return context.WithValue(ctx, contextKeyMultisig, []weave.Condition{MultiSigCondition(id)})
	}

	return context.WithValue(ctx, contextKeyMultisig, append(val, MultiSigCondition(id)))
}

// MultiSigCondition returns condition for a contract ID
func MultiSigCondition(id []byte) weave.Condition {
	return weave.NewCondition("multisig", "usage", id)
}

// Authenticate gets/sets permissions on the given context key
type Authenticate struct {
}

var _ x.Authenticator = Authenticate{}

// GetConditions returns permissions previously set on this context
func (a Authenticate) GetConditions(ctx context.Context) []weave.Condition {
	// (val, ok) form to return nil instead of panic if unset
	val, _ := ctx.Value(contextKeyMultisig).([]weave.Condition)
	if val == nil {
		return nil
	}
	return val
}

// HasAddress returns true iff this address is in GetConditions
func (a Authenticate) HasAddress(ctx context.Context, info weave.BlockInfo, addr weave.Address) bool {
	for _, s := range a.GetConditions(ctx) {
		if addr.Equals(s.Address()) {
			return true
		}
	}
	return false
}
