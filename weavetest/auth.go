package weavetest

import (
	"context"
	"fmt"

	"github.com/iov-one/weave"
)

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
	return addr.Equals(a.Signer.Address())
}

type CtxAuth struct {
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
