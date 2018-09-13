package multisig

import (
	"context"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
)

type contextKey int // local to the hashlock module

const (
	contextKeyMultisig contextKey = iota
)

// withPreimage is a private method, as only this module
// can add a signer
func withMultisig(ctx weave.Context, id []byte) weave.Context {
	return context.WithValue(ctx, contextKeyMultisig, MultiSigCondition(id))
}

// CtxAuther gets/sets permissions on the given context key
type Authenticate struct {
}

var _ x.Authenticator = Authenticate{}

// GetConditions returns permissions previously set on this context
func (a Authenticate) GetConditions(ctx weave.Context) []weave.Condition {
	// (val, ok) form to return nil instead of panic if unset
	val, _ := ctx.Value(contextKeyMultisig).(weave.Condition)
	if val == nil {
		return nil
	}
	return []weave.Condition{val}
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
