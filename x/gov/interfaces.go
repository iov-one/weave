package gov

import (
	weave "github.com/iov-one/weave"
)

// OptionDecoder is needed to parse the raw_options data.
type OptionDecoder func(raw []byte) (weave.Msg, error)

// Executor will do something with the message once it is approved.
type Executor func(ctx weave.Context, store weave.KVStore, msg weave.Msg) (*weave.DeliverResult, error)

// HandlerAsExecutor wraps the msg in a fake Tx to satisfy the Handler interface
// Since a Router and Decorators also expose this interface, we can wrap any stack
// that does not care about the extra Tx info besides Msg.
func HandlerAsExecutor(h weave.Handler) Executor {
	return func(ctx weave.Context, store weave.KVStore, msg weave.Msg) (*weave.DeliverResult, error) {
		tx := &fakeTx{msg: msg}
		return h.Deliver(ctx, store, tx)
	}
}

type fakeTx struct {
	msg weave.Msg
}

var _ weave.Tx = (*fakeTx)(nil)

func (tx fakeTx) GetMsg() (weave.Msg, error) {
	return tx.msg, nil
}

func (tx fakeTx) Marshal() ([]byte, error) {
	return tx.msg.Marshal()
}

func (tx *fakeTx) Unmarshal(data []byte) error {
	// note this will panic if actually run
	return tx.msg.Unmarshal(data)
}
