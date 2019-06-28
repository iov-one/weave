package weave

import (
	"encoding/json"

	abci "github.com/tendermint/tendermint/abci/types"
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
