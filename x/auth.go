package x

import (
	"bytes"

	"github.com/confio/weave"
)

// AuthFunc is a function we can use to extract authentication info
// from the context. This should be passed into the constructor of
// handlers, so we can plug in another authentication system,
// rather than hardcoding x/auth for all extensions.
type AuthFunc func(weave.Context) []weave.Address

// MultiAuth groups together a series of AuthFunc
func MultiAuth(fns ...AuthFunc) AuthFunc {
	return func(ctx weave.Context) (res []weave.Address) {
		for _, fn := range fns {
			add := fn(ctx)
			if len(add) > 0 {
				res = append(res, add...)
			}
		}
		return res
	}
}

// MainSigner returns the first signed if any, otherwise nil
func MainSigner(ctx weave.Context, fn AuthFunc) weave.Address {
	auth := fn(ctx)
	if len(auth) == 0 {
		return nil
	}
	return auth[0]
}

// HasAllSigners returns true if all elements in required are
// also in signed.
func HasAllSigners(required []weave.Address, signed []weave.Address) bool {
	return HasNSigners(len(required), required, signed)
}

// HasSigner returns true if this address has signed
func HasSigner(required weave.Address, signed []weave.Address) bool {
	// simplest....
	for _, signer := range signed {
		if bytes.Equal(required, signer) {
			return true
		}
	}
	return false
}

// HasNSigners returns true if at least n elements in requested are
// also in signed.
// Useful for threshold conditions (1 of 3, 3 of 5, etc...)
func HasNSigners(n int, requested []weave.Address, signed []weave.Address) bool {
	// TODO: Implement when needed
	return false
}
