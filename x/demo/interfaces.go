package demo

import (
	weave "github.com/iov-one/weave"
	weavetest "github.com/iov-one/weave/weavetest"
)

// OptionLoader is needed to parse the raw_options data
type OptionLoader func(raw []byte) (weave.Msg, error)

// Executor will do something with the message once it is approved
type Executor func(ctx weave.Context, store weave.KVStore, msg weave.Msg) (*weave.DeliverResult, error)

// HandlerAsExecutor wraps the msg in a fake Tx to satisfy the Handler interface
func HandlerAsExecutor(h weave.Handler) Executor {
	return func(ctx weave.Context, store weave.KVStore, msg weave.Msg) (*weave.DeliverResult, error) {
		tx := &weavetest.Tx{Msg: msg}
		return h.Deliver(ctx, store, tx)
	}
}
