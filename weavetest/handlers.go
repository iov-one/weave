package weavetest

import (
	"github.com/iov-one/weave"
)

// Handler implements a mock of weave.Handler
//
// Use this handler in your tests. Set XxxResult and XxxErr to control what Xxx
// method call returns. Each method call is counted.
type Handler struct {
	checkCall int
	// CheckResult is returned by Check method.
	CheckResult weave.CheckResult
	// CheckErr if set is returned by Check method.
	CheckErr error

	deliverCall int
	// DeliverResult is returned by Deliver method.
	DeliverResult weave.DeliverResult
	// DeliverErr if set is returned by Deliver method.
	DeliverErr error
}

var _ weave.Handler = (*Handler)(nil)

func (h *Handler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	h.checkCall++
	return h.CheckResult, h.CheckErr
}

func (h *Handler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	h.deliverCall++
	return h.DeliverResult, h.DeliverErr
}

func (h *Handler) CheckCallCount() int {
	return h.checkCall
}

func (h *Handler) DeliverCallCount() int {
	return h.deliverCall
}

func (h *Handler) CallCount() int {
	return h.checkCall + h.deliverCall
}
