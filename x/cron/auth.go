package cron

import (
	"context"

	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
)

type ctxKey int

const (
	ctxKeyConditions ctxKey = iota
)

// withAuth returns a context instance with the conditions attached. Attached
// conditions are used for authentication by authenticator implementation from
// this package.
func withAuth(ctx weave.Context, cs []weave.Condition) weave.Context {
	if old, ok := ctx.Value(ctxKeyConditions).([]weave.Condition); ok {
		cs = append(cs, old...)
	}
	return context.WithValue(ctx, ctxKeyConditions, cs)
}

// Authenticator implements an x.Authenticator interface that should be used to
// authorize cron task execution.
// Use it together with WithAuth function to control the cron task execution
// authentication.
type Authenticator struct{}

var _ x.Authenticator = (*Authenticator)(nil)

// GetConditions implements x.Authenticator interface.
func (Authenticator) GetConditions(ctx weave.Context) []weave.Condition {
	val, ok := ctx.Value(ctxKeyConditions).([]weave.Condition)
	if !ok {
		return nil
	}
	return val
}

// HasAddress implements x.Authenticator interface.
func (a Authenticator) HasAddress(ctx weave.Context, addr weave.Address) bool {
	for _, c := range a.GetConditions(ctx) {
		if addr.Equals(c.Address()) {
			return true
		}
	}
	return false
}
