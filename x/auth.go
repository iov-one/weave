package x

import (
	"github.com/iov-one/weave"
)

// Authenticator is an interface we can use to extract authentication info
// from the context. This should be passed into the constructor of
// handlers, so we can plug in another authentication system,
// rather than hard-coding x/auth for all extensions.
type Authenticator interface {
	// GetConditions reveals all Conditions fulfilled,
	// you may want GetAddresses helper
	GetConditions(weave.Context) []weave.Condition
	// HasAddress checks if any condition matches this address
	HasAddress(weave.Context, weave.Address) bool
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

// GetConditions combines all Conditions from all Authenticators
func (m MultiAuth) GetConditions(ctx weave.Context) []weave.Condition {
	var res []weave.Condition
	for _, impl := range m.impls {
		add := impl.GetConditions(ctx)
		if len(add) > 0 {
			res = append(res, add...)
		}
	}
	// TODO: remove duplicates (don't sort?)
	return res
}

// HasAddress returns true iff any Authenticator support this
func (m MultiAuth) HasAddress(ctx weave.Context, addr weave.Address) bool {
	for _, impl := range m.impls {
		if impl.HasAddress(ctx, addr) {
			return true
		}
	}
	return false
}

// GetAddresses wraps the GetConditions method of any Authenticator
func GetAddresses(ctx weave.Context, auth Authenticator) []weave.Address {
	perms := auth.GetConditions(ctx)
	addrs := make([]weave.Address, len(perms))
	for i, p := range perms {
		addrs[i] = p.Address()
	}
	return addrs
}

// MainSigner returns the first permission if any, otherwise nil
func MainSigner(ctx weave.Context, auth Authenticator) weave.Condition {
	signers := auth.GetConditions(ctx)
	if len(signers) == 0 {
		return nil
	}
	return signers[0]
}

// HasAllAddresses returns true if all elements in required are
// also in context.
func HasAllAddresses(ctx weave.Context, auth Authenticator, required []weave.Address) bool {
	for _, r := range required {
		if !auth.HasAddress(ctx, r) {
			return false
		}
	}
	return true
}

// HasNAddresses returns true if at least n elements in requested are
// also in context.
func HasNAddresses(ctx weave.Context, auth Authenticator, required []weave.Address, n int) bool {
	// Special case: is this an error???
	if n <= 0 {
		return true
	}

	for _, r := range required {
		if auth.HasAddress(ctx, r) {
			n--
			if n == 0 {
				return true
			}
		}
	}
	return false
}

// HasAllConditions returns true if all elements in required are
// also in context.
func HasAllConditions(ctx weave.Context, auth Authenticator, required []weave.Condition) bool {
	return HasNConditions(ctx, auth, required, len(required))
}

// HasNConditions returns true if at least n elements in requested are
// also in context.
// Useful for threshold conditions (1 of 3, 3 of 5, etc...)
func HasNConditions(ctx weave.Context, auth Authenticator, requested []weave.Condition, n int) bool {
	// Special case: is this an error???
	if n <= 0 {
		return true
	}
	perms := auth.GetConditions(ctx)
	// NOTE: optimize this with sort from N^2 to N*log N (?)
	// low-priority, as N is always small, better that it works
	for _, perm := range requested {
		if hasPerm(perms, perm) {
			n--
			if n == 0 {
				return true
			}
		}
	}
	return false
}

func hasPerm(perms []weave.Condition, perm weave.Condition) bool {
	for _, p := range perms {
		if p.Equals(perm) {
			return true
		}
	}
	return false
}
