package weavetest

import "github.com/iov-one/weave"

type Handler struct {
	checkCall   int
	CheckResult weave.CheckResult
	CheckErr    error

	deliverCall   int
	DeliverResult weave.DeliverResult
	DeliverErr    error
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
