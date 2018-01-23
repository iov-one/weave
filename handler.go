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
