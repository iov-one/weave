package utils

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

// Recovery is a decorator to recover from panics in transactions,
// so we can log them as errors
type Recovery struct{}

var _ weave.Decorator = Recovery{}

// NewRecovery creates a Recovery decorator
func NewRecovery() Recovery {
	return Recovery{}
}

// Check turns panics into normal errors
func (r Recovery) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (_ *weave.CheckResult, err error) {
	defer errors.Recover(&err)
	return next.Check(ctx, store, tx)
}

// Deliver turns panics into normal errors
func (r Recovery) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (_ *weave.DeliverResult, err error) {
	defer errors.Recover(&err)
	return next.Deliver(ctx, store, tx)
}
