package app

import "github.com/confio/weave"

//-------------- counting -------------------------

// countingDecorator checks if it is called, once down, once out
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

// countingHandler checks if it is called
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
