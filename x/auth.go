package x

import (
	"github.com/confio/weave"
)

// Authenticator is an interface we can use to extract authentication info
// from the context. This should be passed into the constructor of
// handlers, so we can plug in another authentication system,
// rather than hardcoding x/auth for all extensions.
type Authenticator interface {
	GetPermissions(weave.Context) []weave.Address
	HasPermission(weave.Context, weave.Address) bool
}

// MultiAuth chains together many Authenticators into one
type MultiAuth struct {
	impls []Authenticator
}

var _ Authenticator = MultiAuth{}

// ChainAuth groups together a series of Authenticator
func ChainAuth(impls ...Authenticator) MultiAuth {
	return MultiAuth{impls}
}

// GetPermissions combines all Permissions from all Authenenticators
func (m MultiAuth) GetPermissions(ctx weave.Context) []weave.Address {
	var res []weave.Address
	for _, impl := range m.impls {
		add := impl.GetPermissions(ctx)
		if len(add) > 0 {
			res = append(res, add...)
		}
	}
	// TODO: remove duplicates (don't sort?)
	return res
}

// HasPermission returns true iff any Authenticator support this
func (m MultiAuth) HasPermission(ctx weave.Context, addr weave.Address) bool {
	for _, impl := range m.impls {
		if impl.HasPermission(ctx, addr) {
			return true
		}
	}
	return false
}

// MainSigner returns the first signed if any, otherwise nil
func MainSigner(ctx weave.Context, auth Authenticator) weave.Address {
	signers := auth.GetPermissions(ctx)
	if len(signers) == 0 {
		return nil
	}
	return signers[0]
}

// HasAllSigners returns true if all elements in required are
// also in signed.
func HasAllSigners(ctx weave.Context, auth Authenticator, required []weave.Address) bool {
	return HasNSigners(ctx, auth, required, len(required))
}

// HasNSigners returns true if at least n elements in requested are
// also in signed.
// Useful for threshold conditions (1 of 3, 3 of 5, etc...)
func HasNSigners(ctx weave.Context, auth Authenticator, requested []weave.Address, n int) bool {
	// Special case: is this an error???
	if n <= 0 {
		return true
	}
	// check requested until enough found, or all checked
	for _, addr := range requested {
		if auth.HasPermission(ctx, addr) {
			n--
			if n == 0 {
				return true
			}
		}
	}
	return false
}
