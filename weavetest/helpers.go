package weavetest

/*
import (
	"context"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/crypto"
	"github.com/tendermint/tendermint/libs/common"
)

//--------------- expose helpers -----

// TestHelpers returns helper objects for tests,
// encapsulated in one object to be easily imported in other packages
type TestHelpers struct{}



// WriteHandler will write the given key/value pair to the KVStore,
// and return the error (use nil for success)
func (TestHelpers) WriteHandler(key, value []byte, err error) weave.Handler {
	return writeHandler{
		key:   key,
		value: value,
		err:   err,
	}
}

// WriteDecorator will write the given key/value pair to the KVStore,
// either before or after calling down the stack.
// Returns (res, err) from child handler untouched
func (TestHelpers) WriteDecorator(key, value []byte, after bool) weave.Decorator {
	return writeDecorator{
		key:   key,
		value: value,
		after: after,
	}
}

// TagHandler writes a tag to DeliverResult and returns error of nil
// returns error, but doens't write any tags on CheckTx
func (TestHelpers) TagHandler(key, value []byte, err error) weave.Handler {
	return tagHandler{
		key:   key,
		value: value,
		err:   err,
	}
}

// Wrap wraps the handler with one decorator and returns it
// as a single handler.
// Minimal version of ChainDecorators for test cases
func (TestHelpers) Wrap(d weave.Decorator, h weave.Handler) weave.Handler {
	return wrappedHandler{
		d: d,
		h: h,
	}
}



// Authenticate returns an Authenticator that gives permissions
// to the given addresses
func (TestHelpers) Authenticate(perms ...weave.Condition) Authenticator {
	return mockAuth{perms}
}

// CtxAuth returns an authenticator that uses the context
// getting and setting with the given key
func (TestHelpers) CtxAuth(key interface{}) CtxAuther {
	return CtxAuther{key}
}


//------ static auth (added in constructor)

type mockAuth struct {
	signers []weave.Condition
}

var _ Authenticator = mockAuth{}

func (a mockAuth) GetConditions(weave.Context) []weave.Condition {
	return a.signers
}

func (a mockAuth) HasAddress(ctx weave.Context, addr weave.Address) bool {
	for _, s := range a.signers {
		if addr.Equals(s.Address()) {
			return true
		}
	}
	return false
}

//----- dynamic auth (based on ctx)

// CtxAuther gets/sets permissions on the given context key
type CtxAuther struct {
	key interface{}
}

var _ Authenticator = CtxAuther{}

// SetConditions returns a context with the given permissions set
func (a CtxAuther) SetConditions(ctx weave.Context, perms ...weave.Condition) weave.Context {
	return context.WithValue(ctx, a.key, perms)
}

// GetConditions returns permissions previously set on this context
func (a CtxAuther) GetConditions(ctx weave.Context) []weave.Condition {
	val, _ := ctx.Value(a.key).([]weave.Condition)
	return val
}

// HasAddress returns true iff this address is in GetConditions
func (a CtxAuther) HasAddress(ctx weave.Context, addr weave.Address) bool {
	for _, s := range a.GetConditions(ctx) {
		if addr.Equals(s.Address()) {
			return true
		}
	}
	return false
}


//----------------- writers --------

// writeHandler writes the key, value pair and returns the error (may be nil)
type writeHandler struct {
	key   []byte
	value []byte
	err   error
}

var _ weave.Handler = writeHandler{}

func (h writeHandler) Check(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.CheckResult, error) {

	store.Set(h.key, h.value)
	return weave.CheckResult{}, h.err
}

func (h writeHandler) Deliver(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.DeliverResult, error) {

	store.Set(h.key, h.value)
	return weave.DeliverResult{}, h.err
}

// writeDecorator writes the key, value pair.
// either before or after calling the handlers
type writeDecorator struct {
	key   []byte
	value []byte
	after bool
}

var _ weave.Decorator = writeDecorator{}

func (d writeDecorator) Check(ctx weave.Context, store weave.KVStore,
	tx weave.Tx, next weave.Checker) (weave.CheckResult, error) {

	if !d.after {
		store.Set(d.key, d.value)
	}
	res, err := next.Check(ctx, store, tx)
	if d.after && err == nil {
		store.Set(d.key, d.value)
	}
	return res, err
}

func (d writeDecorator) Deliver(ctx weave.Context, store weave.KVStore,
	tx weave.Tx, next weave.Deliverer) (weave.DeliverResult, error) {

	if !d.after {
		store.Set(d.key, d.value)
	}
	res, err := next.Deliver(ctx, store, tx)
	if d.after && err == nil {
		store.Set(d.key, d.value)
	}
	return res, err
}

//----------------- misc --------

// tagHandler writes the key, value pair and returns the error (may be nil)
type tagHandler struct {
	key   []byte
	value []byte
	err   error
}

var _ weave.Handler = tagHandler{}

func (h tagHandler) Check(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.CheckResult, error) {
	return weave.CheckResult{}, h.err
}

func (h tagHandler) Deliver(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.DeliverResult, error) {

	tags := common.KVPairs{{Key: h.key, Value: h.value}}
	return weave.DeliverResult{Tags: tags}, h.err
}

type wrappedHandler struct {
	d weave.Decorator
	h weave.Handler
}

var _ weave.Handler = wrappedHandler{}

func (w wrappedHandler) Check(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.CheckResult, error) {

	return w.d.Check(ctx, store, tx, w.h)
}

func (w wrappedHandler) Deliver(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.DeliverResult, error) {

	return w.d.Deliver(ctx, store, tx, w.h)
}

type EnumHelpers struct{}

func (EnumHelpers) AsList(enum map[string]int32) []string {
	res := make([]string, 0, len(enum))
	for name := range enum {
		res = append(res, name)
	}

	return res
}
*/
