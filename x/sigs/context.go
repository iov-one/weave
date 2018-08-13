package sigs

import (
	"context"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
)

//------------------- Context --------
// Add context information specific to this package

type contextKey int // local to the auth module

const (
	contextKeySigners contextKey = iota
)

// withSigners is a private method, as only this module
// can add a signer
func withSigners(ctx weave.Context, signers []weave.Condition) weave.Context {
	return context.WithValue(ctx, contextKeySigners, signers)
}

// Authenticate implements x.Authenticator and provides
// authentication based on public-key signatures.
type Authenticate struct{}

var _ x.Authenticator = Authenticate{}

// GetConditions returns who signed the current Context.
// May be empty
func (a Authenticate) GetConditions(ctx weave.Context) []weave.Condition {
	// (val, ok) form to return nil instead of panic if unset
	val, _ := ctx.Value(contextKeySigners).([]weave.Condition)
	// if we were paranoid about our own code, we would deep-copy
	// the signers here
	return val
}

// HasAddress returns true if the given address
// had signed in the current Context.
func (a Authenticate) HasAddress(ctx weave.Context, addr weave.Address) bool {
	signers := a.GetConditions(ctx)
	for _, s := range signers {
		if addr.Equals(s.Address()) {
			return true
		}
	}
	return false
}
