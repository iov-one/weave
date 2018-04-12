package hashlock

import (
	"context"

	"github.com/confio/weave"
	"github.com/confio/weave/x"
)

//------------------- Context --------
// Add context information specific to this package

type contextKey int // local to the hashlock module

const (
	contextKeyPreimage contextKey = iota
)

// withPreimage is a private method, as only this module
// can add a signer
func withPreimage(ctx weave.Context, preimage []byte) weave.Context {
	return context.WithValue(ctx, contextKeyPreimage,
		PreimagePermission(preimage))
}

// Authenticate implements x.Authenticator and provides
// authentication based on public-key signatures.
type Authenticate struct{}

var _ x.Authenticator = Authenticate{}

// GetPermissions returns which preimages have authorized the current Context.
// May be nil
func (a Authenticate) GetPermissions(ctx weave.Context) []weave.Permission {
	// (val, ok) form to return nil instead of panic if unset
	val, _ := ctx.Value(contextKeyPreimage).(weave.Permission)
	if val == nil {
		return nil
	}
	return []weave.Permission{val}
}

// HasAddress returns true if the given address
// had the preimage permission in the current Context.
func (a Authenticate) HasAddress(ctx weave.Context, addr weave.Address) bool {
	val, _ := ctx.Value(contextKeyPreimage).(weave.Permission)
	if val != nil && val.Address().Equals(addr) {
		return true
	}
	return false
}
