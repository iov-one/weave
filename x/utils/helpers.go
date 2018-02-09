package utils

import "github.com/confio/weave"

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

// Counting handler returns success and counts times called
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

func (panicHandler) Check(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.CheckResult, error) {

	panic("fire alarm!!!")
}

func (panicHandler) Deliver(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.DeliverResult, error) {

	panic("fire alarm!!!")
}
