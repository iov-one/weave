package weavetest

import (
	"context"
	"fmt"

	"github.com/iov-one/weave"
)

// Auth is a mock implementing x.Authenticator interface.
//
// This structure authenticates any of referenced conditions.
// You can use either Signer or Signers (or both) attributes to reference
// conditions. This is for the convinience and each time all signers
// (regardless which attribute) are considered.
type Auth struct {
	// Signer represents an authentication of a single signer. This is a
	// convinience attribute when creating an authentication method for a
	// single signer.
	// When authenticating all signers declared on this structure are
	// considered.
	Signer weave.Condition

	// Signers represents an authentication of multiple signers.
	Signers []weave.Condition
}

func (a *Auth) GetConditions(weave.Context) []weave.Condition {
	if a.Signer != nil {
		return append(a.Signers, a.Signer)
	}
	return a.Signers
}

func (a *Auth) HasAddress(ctx weave.Context, addr weave.Address) bool {
	for _, s := range a.Signers {
		if addr.Equals(s.Address()) {
			return true
		}
	}
	if a.Signer == nil {
		return false
	}
	return addr.Equals(a.Signer.Address())
}

// CtxAuth is a mock implementing x.Authenticator interface.
//
// This implementation is using context to store and retrieve permissions.
type CtxAuth struct {
	// Key used to set and retrieve conditions from the context. For
	// convinience only string type keys are allowed.
	Key string
}

func (a *CtxAuth) SetConditions(ctx weave.Context, permissions ...weave.Condition) weave.Context {
	return context.WithValue(ctx, a.Key, permissions)
}

func (a *CtxAuth) GetConditions(ctx weave.Context) []weave.Condition {
	val := ctx.Value(a.Key)
	if val == nil {
		return nil
	}
	conds, ok := val.([]weave.Condition)
	if !ok {
		panic(fmt.Sprintf("instead of []weave.Condition got %T", ctx.Value(a.Key)))
	}
	return conds
}

func (a *CtxAuth) HasAddress(ctx weave.Context, addr weave.Address) bool {
	for _, s := range a.GetConditions(ctx) {
		if addr.Equals(s.Address()) {
			return true
		}
	}
	return false
}
