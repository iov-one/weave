package weave

import (
	"bytes"
	"encoding/json"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/iov-one/weave/errors"
)

// Handler is a core engine that can process a few specific messages
// This could represent "coin transfer", or "bonding stake to a validator"
type Handler interface {
	Checker
	Deliverer
}

// Checker is a subset of Handler to verify the validity of a transaction.
// It is its own interface to allow better type controls in the next
// arguments in Decorator
type Checker interface {
	Check(ctx Context, store KVStore, tx Tx) (*CheckResult, error)
}

// Deliverer is a subset of Handler to execute a transaction.
// It is its own interface to allow better type controls in the next
// arguments in Decorator
type Deliverer interface {
	Deliver(ctx Context, store KVStore, tx Tx) (*DeliverResult, error)
}

// Decorator wraps a Handler to provide common functionality
// like authentication, or fee-handling, to many Handlers
type Decorator interface {
	Check(ctx Context, store KVStore, tx Tx, next Checker) (*CheckResult, error)
	Deliver(ctx Context, store KVStore, tx Tx, next Deliverer) (*DeliverResult, error)
}

// Registry is an interface to register your handler,
// the setup side of a Router
type Registry interface {
	// Handle assigns given handler to handle processing of every message
	// of provided type.
	// Using a message with an invalid path panics.
	// Registering a handler for a message more than ones panics.
	Handle(Msg, Handler)
}

// Options are the app options
// Each extension can look up it's key and parse the json as desired
type Options map[string]json.RawMessage

// ReadOptions reads the values stored under a given key,
// and parses the json into the given obj.
// Returns an error if it cannot parse.
// Noop and no error if key is missing
func (o Options) ReadOptions(key string, obj interface{}) error {
	msg := o[key]
	if len(msg) == 0 {
		return nil
	}
	return json.Unmarshal(msg, obj)
}

// Stream expects an array of json elements and allows to process them sequentially
// this helps when one needs to parse a large json without having any memory leaks.
// Returns ErrEmpty on empty key or when there are no more elements.
// Returns ErrState when the stream has finished/encountered a Decode error.
func (o Options) Stream(key string) (func(obj interface{}) error, error) {
	msg := o[key]
	if len(msg) == 0 {
		return nil, errors.Wrap(errors.ErrEmpty, "data")
	}

	dec := json.NewDecoder(bytes.NewReader(msg))

	// read opening bracket
	if _, err := dec.Token(); err != nil {
		return nil, errors.Wrapf(errors.ErrInput, "opening bracket %s", err)

	}

	closed := false

	return func(obj interface{}) error {
		if closed {
			return errors.Wrap(errors.ErrState, "closed")
		}

		if dec.More() {
			if err := dec.Decode(obj); err != nil {
				return errors.Wrapf(errors.ErrInput, "decode %s", err)
			}
			return nil
		}

		closed = true
		// read closing bracket
		if _, err := dec.Token(); err != nil {
			return errors.Wrapf(errors.ErrInput, "closing bracket %s", err)
		}

		return errors.Wrap(errors.ErrEmpty, "end")
	}, nil

}

// GenesisParams represents parameters set in genesis that could be useful
// for some of the extensions.
type GenesisParams struct {
	Validators []abci.ValidatorUpdate
}

// FromInitChain initialises GenesisParams using abci.RequestInitChain
// data.
func FromInitChain(req abci.RequestInitChain) GenesisParams {
	return GenesisParams{
		Validators: req.Validators,
	}
}

// Initializer implementations are used to initialize
// extensions from genesis file contents
type Initializer interface {
	FromGenesis(opts Options, params GenesisParams, kv KVStore) error
}
