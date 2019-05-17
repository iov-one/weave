package gov

import (
	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	weavetest "github.com/iov-one/weave/weavetest"
)

// OptionDecoder is needed to parse the raw_options data
type OptionDecoder func(raw []byte) (weave.Msg, error)

// ProtoSum is a basic interface for a protobuf model containing a Sum (oneof) type
// Given this, we can auto-generate an OptionDecoder
type ProtoSum interface {
	Unmarshal([]byte) error
	GetSum() interface{}
}

// LoadFromProtoSum will generate an OptionDecoder from a protobuf object with a Sum oneof type
// where all elements of the oneof can be cast to weave.Msg
func LoadFromProtoSum(model ProtoSum) OptionDecoder {
	return func(raw []byte) (weave.Msg, error) {
		err := model.Unmarshal(raw)
		if err != nil {
			return nil, errors.Wrap(err, "cannot parse data into Options struct")
		}
		return weave.ExtractMsgFromSum(model.GetSum())
	}
}

// Executor will do something with the message once it is approved
type Executor func(ctx weave.Context, store weave.KVStore, msg weave.Msg) (*weave.DeliverResult, error)

// HandlerAsExecutor wraps the msg in a fake Tx to satisfy the Handler interface
// Since a Router and Decorators also expose this interface, we can wrap any stack
// that does not care about the extra Tx info besides Msg
func HandlerAsExecutor(h weave.Handler) Executor {
	return func(ctx weave.Context, store weave.KVStore, msg weave.Msg) (*weave.DeliverResult, error) {
		tx := &weavetest.Tx{Msg: msg}
		return h.Deliver(ctx, store, tx)
	}
}
