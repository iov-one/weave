package app

import (
	"reflect"

	"github.com/iov-one/weave"
)

// Decorators holds a chain of decorators, not yet resolved by a Handler
type Decorators struct {
	chain []weave.Decorator
}

/*
ChainDecorators takes a chain of decorators,
and upon adding a final Handler (often a Router),
returns a Handler that will execute this whole stack.

  app.ChainDecorators(
    util.NewLogging(),
    util.NewRecovery(),
    auth.NewDecorator(),
    coins.NewFeeDecorator(),
    util.NewSavepoint().OnDeliver(),
  ).WithHandler(
    myapp.NewRouter(),
  )
*/
func ChainDecorators(chain ...weave.Decorator) Decorators {
	chain = cutoffNil(chain)
	return Decorators{}.Chain(chain...)
}

// Chain allows us to keep adding more Decorators to the chain
func (d Decorators) Chain(chain ...weave.Decorator) Decorators {
	chain = cutoffNil(chain)
	newChain := append(d.chain, chain...)
	return Decorators{newChain}
}

// cutoffNil will in-place remove all all nil values from given slice.
func cutoffNil(ds []weave.Decorator) []weave.Decorator {
	var cutoff int
	for i := 0; i < len(ds); i++ {
		ds[i-cutoff] = ds[i]
		if ds[i] == nil || (reflect.ValueOf(ds[i]).Kind() == reflect.Ptr && reflect.ValueOf(ds[i]).IsNil()) {
			cutoff++
		}
	}
	return ds[:len(ds)-cutoff]
}

// WithHandler resolves the stack and returns a concrete Handler
// that will pass through the chain of decorators before calling
// the final Handler.
func (d Decorators) WithHandler(h weave.Handler) weave.Handler {
	// start wrapping the handler from last decorator to first one
	// as the top of the chain is understood to be executed first
	for i := len(d.chain) - 1; i >= 0; i-- {
		h = step{d: d.chain[i], next: h}
	}
	return h
}

//------------------ internal types to build chain ---------------

// step captures one step executing a decorator around a
// specific Handler. Simplified version of a closure.
//
// Heavily inspired by negroni's design
type step struct {
	d    weave.Decorator
	next weave.Handler
}

var _ weave.Handler = step{}

// Check passes the handler into the decorator, implements Handler
func (s step) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	return s.d.Check(ctx, store, tx, s.next)
}

// Deliver passes the handler into the decorator, implements Handler
func (s step) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	return s.d.Deliver(ctx, store, tx, s.next)
}
