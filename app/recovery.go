package app

import (
	"fmt"

	"github.com/confio/weave"
	"github.com/confio/weave/errors"
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
func (r Recovery) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Checker) (res weave.CheckResult, err error) {

	defer func() {
		if r := recover(); r != nil {
			err = normalizePanic(r)
		}
	}()
	return next.Check(ctx, store, tx)
}

// Deliver turns panics into normal errors
func (r Recovery) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Deliverer) (res weave.DeliverResult, err error) {

	defer func() {
		if r := recover(); r != nil {
			err = normalizePanic(r)
		}
	}()
	return next.Deliver(ctx, store, tx)
}

// normalizePanic makes sure we can get a nice TMError (with stack) out of it
func normalizePanic(p interface{}) error {
	if err, isErr := p.(error); isErr {
		return errors.Wrap(err)
	}
	msg := fmt.Sprintf("%v", p)
	return errors.ErrInternal(msg)
}
