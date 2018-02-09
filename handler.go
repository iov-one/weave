package weave

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
	Check(ctx Context, store KVStore, tx Tx) (CheckResult, error)
}

// Checker is a subset of Handler to execute a transaction.
// It is its own interface to allow better type controls in the next
// arguments in Decorator
type Deliverer interface {
	Deliver(ctx Context, store KVStore, tx Tx) (DeliverResult, error)
}

// Decorator wraps a Handler to provide common functionality
// like authentication, or fee-handling, to many Handlers
type Decorator interface {
	Check(ctx Context, store KVStore, tx Tx, next Checker) (CheckResult, error)
	Deliver(ctx Context, store KVStore, tx Tx, next Deliverer) (DeliverResult, error)
}

// Ticker is a method that is called the beginning of every block,
// which can be used to perform periodic or delayed tasks
type Ticker interface {
	Tick(ctx Context, store KVStore) (TickResult, error)
}

// Registry is an interface to register your handler,
// the setup side of a Router
type Registry interface {
	Handle(path string, h Handler)
}

// Options loads the options stored under key into the
// given interface using reflection
type Options interface {
	ReadOptions(key string, v interface{}) error
}

// InitStater implementations are used to initialize
// extensions from genesis file contents
type InitStater interface {
	InitState(Options, KVStore) error
}
