package weave

import (
	"fmt"
)

const (
	// KeyQueryMod means to query for exact match (key)
	KeyQueryMod = ""
	// PrefixQueryMod means to query for anything with this prefix
	PrefixQueryMod = "prefix"
	// RangeQueryMod means to expect complex range query
	// TODO: implement
	RangeQueryMod = "range"
)

// Model groups together key and value to return
type Model struct {
	Key   []byte
	Value []byte
}

// Pair constructs a model from a key-value pair
func Pair(key, value []byte) Model {
	return Model{
		Key:   key,
		Value: value,
	}
}

// QueryHandler is anything that can process ABCI queries
type QueryHandler interface {
	Query(db ReadOnlyKVStore, mod string, data []byte) ([]Model, error)
}

// QueryRegister is a function that adds some handlers
// to this router
type QueryRegister func(QueryRouter)

// QueryRouter allows us to register many query handlers
// to different paths and then direct each query
// to the proper handler.
//
// Minimal interface modeled after net/http.ServeMux
type QueryRouter struct {
	routes map[string]QueryHandler
}

// NewQueryRouter initializes a QueryRouter with no routes
func NewQueryRouter() QueryRouter {
	return QueryRouter{
		routes: make(map[string]QueryHandler, 10),
	}
}

// RegisterAll registers a number of QueryRegister at once
func (r QueryRouter) RegisterAll(qr ...QueryRegister) {
	for _, q := range qr {
		q(r)
	}
}

// Register adds a new Handler for the given path. This function panics if a
// handler for given path is already registered.
//
// Path should be constructed using following rules:
// - always use plural form of the model name it represents (unless uncountable)
// - use only lower case characters, no numbers, no underscore, dash or any
//   other special characters
// For example, path for the UserProfile model handler is "userprofiles".
func (r QueryRouter) Register(path string, h QueryHandler) {
	if _, ok := r.routes[path]; ok {
		panic(fmt.Sprintf("Re-registering route: %s", path))
	}
	r.routes[path] = h
}

// Handler returns the registered Handler for this path.
// If no path is found, returns a noSuchPath Handler
// Always returns a non-nil Handler
func (r QueryRouter) Handler(path string) QueryHandler {
	return r.routes[path]
}
