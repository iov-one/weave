package sigs

import (
	"context"

	"github.com/confio/weave"
	"github.com/confio/weave/x"
)

//------------------- Context --------
// Add context information specific to this package

type contextKey int // local to the auth module

const (
	contextKeySigners contextKey = iota
)

// withSigners is a private method, as only this module
// can add a signer
func withSigners(ctx weave.Context, signers []weave.Address) weave.Context {
	return context.WithValue(ctx, contextKeySigners, signers)
}

type Authenticate struct{}

var _ x.Authenticator = Authenticate{}

// GetPermissions returns who signed the current Context.
// May be empty
func (a Authenticate) GetPermissions(ctx weave.Context) []weave.Address {
	// (val, ok) form to return nil instead of panic if unset
	val, _ := ctx.Value(contextKeySigners).([]weave.Address)
	// if we were paranoid about our own code, we would deep-copy
	// the signers here
	return val
}

// HasPermission returns who signed the current Context.
// May be empty
func (a Authenticate) HasPermission(ctx weave.Context, addr weave.Address) bool {
	signers := a.GetPermissions(ctx)
	for _, s := range signers {
		if addr.Equals(s) {
			return true
		}
	}
	return false
}
