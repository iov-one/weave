package weavetest

import "github.com/iov-one/weave"

// Decorator is a mock implementation of the weave.Decorator interface.
//
// Set CheckErr or DeliverErr to force error response for corresponding method.
// If error attributes are not set then wrapped handler method is called and
// its result returned.
// Each method call is counted. Regardless of the method call result the
// counter is incremented.
type Decorator struct {
	checkCall int
	// CheckErr if set is returned by the Check method before calling
	// the wrapped handler.
	CheckErr error

	deliverCall int
	// DeliverErr if set is returned by the Deliver method before calling
	// the wrapped handler.
	DeliverErr error
}

var _ weave.Decorator = (*Decorator)(nil)

func (d *Decorator) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx, next weave.Checker) (*weave.CheckResult, error) {
	d.checkCall++

	if d.CheckErr != nil {
		return &weave.CheckResult{}, d.CheckErr
	}
	return next.Check(ctx, db, tx)
}

func (d *Decorator) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx, next weave.Deliverer) (*weave.DeliverResult, error) {
	d.deliverCall++

	if d.DeliverErr != nil {
		return &weave.DeliverResult{}, d.DeliverErr
	}
	return next.Deliver(ctx, db, tx)
}

func (d *Decorator) CheckCallCount() int {
	return d.checkCall
}

func (d *Decorator) DeliverCallCount() int {
	return d.deliverCall
}

func (d *Decorator) CallCount() int {
	return d.checkCall + d.deliverCall
}

func Decorate(h weave.Handler, d weave.Decorator) weave.Handler {
	return &decoratedHandler{hn: h, dc: d}
}

type decoratedHandler struct {
	hn weave.Handler
	dc weave.Decorator
}

var _ weave.Handler = (*decoratedHandler)(nil)

func (d *decoratedHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	return d.dc.Check(ctx, db, tx, d.hn)
}

func (d *decoratedHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	return d.dc.Deliver(ctx, db, tx, d.hn)
}
