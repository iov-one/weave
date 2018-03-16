package x

import (
	"github.com/confio/weave"
	"github.com/confio/weave/crypto"
)

//--------------- expose helpers -----

// TestHelpers returns helper objects for tests,
// encapsulated in one object to be easily imported in other packages
type TestHelpers struct{}

// CountingDecorator passes tx along, and counts how many times it was called.
// Adds one on input down, one on output up,
// to differentiate panic from error
func (TestHelpers) CountingDecorator() CountingDecorator {
	return &countingDecorator{}
}

// CountingHandler returns success and counts times called
func (TestHelpers) CountingHandler() CountingHandler {
	return &countingHandler{}
}

// ErrorDecorator always returns the given error when called
func (TestHelpers) ErrorDecorator(err error) weave.Decorator {
	return errorDecorator{err}
}

// ErrorHandler always returns the given error when called
func (TestHelpers) ErrorHandler(err error) weave.Handler {
	return errorHandler{err}
}

// PanicAtHeightDecorator will panic if ctx.height >= h
func (TestHelpers) PanicAtHeightDecorator(h int64) weave.Decorator {
	return panicAtHeightDecorator{h}
}

// PanicHandler always pancis with the given error when called
func (TestHelpers) PanicHandler(err error) weave.Handler {
	return panicHandler{err}
}

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

// MakeKey returns a random PrivateKey and the associated address
func (TestHelpers) MakeKey() (crypto.Signer, weave.Address) {
	priv := crypto.GenPrivKeyEd25519()
	addr := priv.PublicKey().Address()
	return priv, addr
}

// MockMsg returns a weave.Msg object holding these bytes
func (TestHelpers) MockMsg(bz []byte) weave.Msg {
	return &mockMsg{bz}
}

// MockTx returns a minimal weave.Tx object holding this Msg
func (TestHelpers) MockTx(msg weave.Msg) weave.Tx {
	return &mockTx{msg}
}

// Authenticate returns an Authenticator that gives permissions
// to the given addresses
func (TestHelpers) Authenticate(addrs ...weave.Address) Authenticator {
	return mockAuth{addrs}
}

// CountingDecorator keeps track of number of times called.
// 2x per call, 1x per call with panic inside
type CountingDecorator interface {
	GetCount() int
	weave.Decorator
}

// CountingHandler keeps track of number of times called.
// 1x per call
type CountingHandler interface {
	GetCount() int
	weave.Handler
}

//--------------- tx and msg -----------------------

//------ msg
type mockMsg struct {
	data []byte
}

var _ weave.Msg = (*mockMsg)(nil)

func (m mockMsg) Marshal() ([]byte, error) {
	return m.data, nil
}

func (m *mockMsg) Unmarshal(bz []byte) error {
	m.data = bz
	return nil
}

func (m mockMsg) Path() string {
	return "mock"
}

//------ tx
type mockTx struct {
	msg weave.Msg
}

var _ weave.Tx = (*mockTx)(nil)

func (m mockTx) GetMsg() (weave.Msg, error) {
	return m.msg, nil
}

func (m mockTx) Marshal() ([]byte, error) {
	return m.msg.Marshal()
}

func (m *mockTx) Unmarshal(bz []byte) error {
	return m.msg.Unmarshal(bz)
}

// res, err := msg.Marshal()
// assert.NoError(t, err)
// assert.Equal(t, bz, res)
// tm, err := tx.GetMsg()
// assert.NoError(t, err)
// assert.Equal(t, msg, tm)

//------ auth

type mockAuth struct {
	signers []weave.Address
}

var _ Authenticator = mockAuth{}

func (a mockAuth) GetPermissions(weave.Context) []weave.Address {
	return a.signers
}

func (a mockAuth) HasPermission(ctx weave.Context, addr weave.Address) bool {
	for _, s := range a.signers {
		if addr.Equals(s) {
			return true
		}
	}
	return false
}

//-------------- counting -------------------------

type countingDecorator struct {
	called int
}

var _ weave.Decorator = (*countingDecorator)(nil)

func (c *countingDecorator) Check(ctx weave.Context, store weave.KVStore,
	tx weave.Tx, next weave.Checker) (weave.CheckResult, error) {

	c.called++
	res, err := next.Check(ctx, store, tx)
	c.called++
	return res, err
}

func (c *countingDecorator) Deliver(ctx weave.Context, store weave.KVStore,
	tx weave.Tx, next weave.Deliverer) (weave.DeliverResult, error) {

	c.called++
	res, err := next.Deliver(ctx, store, tx)
	c.called++
	return res, err
}

func (c *countingDecorator) GetCount() int {
	return c.called
}

// countingHandler counts how many times it was called
type countingHandler struct {
	called int
}

var _ weave.Handler = (*countingHandler)(nil)

func (c *countingHandler) Check(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.CheckResult, error) {

	c.called++
	return weave.CheckResult{}, nil
}

func (c *countingHandler) Deliver(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.DeliverResult, error) {

	c.called++
	return weave.DeliverResult{}, nil
}

func (c *countingHandler) GetCount() int {
	return c.called
}

//----------- errors ------------

// errorDecorator returns the given error
type errorDecorator struct {
	err error
}

var _ weave.Decorator = errorDecorator{}

func (e errorDecorator) Check(ctx weave.Context, store weave.KVStore,
	tx weave.Tx, next weave.Checker) (weave.CheckResult, error) {

	return weave.CheckResult{}, e.err
}

func (e errorDecorator) Deliver(ctx weave.Context, store weave.KVStore,
	tx weave.Tx, next weave.Deliverer) (weave.DeliverResult, error) {

	return weave.DeliverResult{}, e.err
}

// errorHandler returns the given error
type errorHandler struct {
	err error
}

var _ weave.Handler = errorHandler{}

func (e errorHandler) Check(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.CheckResult, error) {

	return weave.CheckResult{}, e.err
}

func (e errorHandler) Deliver(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.DeliverResult, error) {

	return weave.DeliverResult{}, e.err
}

// panicAtHeightDecorator panics if ctx.height >= p.height
type panicAtHeightDecorator struct {
	height int64
}

var _ weave.Decorator = panicAtHeightDecorator{}

func (p panicAtHeightDecorator) Check(ctx weave.Context, store weave.KVStore,
	tx weave.Tx, next weave.Checker) (weave.CheckResult, error) {

	if val, _ := weave.GetHeight(ctx); val > p.height {
		panic("too high")
	}
	return next.Check(ctx, store, tx)
}

func (p panicAtHeightDecorator) Deliver(ctx weave.Context, store weave.KVStore,
	tx weave.Tx, next weave.Deliverer) (weave.DeliverResult, error) {

	if val, _ := weave.GetHeight(ctx); val > p.height {
		panic("too high")
	}
	return next.Deliver(ctx, store, tx)
}

// panicHandler always panics
type panicHandler struct {
	err error
}

var _ weave.Handler = panicHandler{}

func (p panicHandler) Check(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.CheckResult, error) {

	panic(p.err)
}

func (p panicHandler) Deliver(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.DeliverResult, error) {

	panic(p.err)
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
	if d.after {
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
	if d.after {
		store.Set(d.key, d.value)
	}
	return res, err
}
