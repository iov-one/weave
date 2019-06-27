package app

import (
	"fmt"
	"regexp"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

// isPath is the RegExp to ensure the routes make sense
var isPath = regexp.MustCompile(`^[a-zA-Z0-9_/]+$`).MatchString

// Router allows us to register many handlers with different
// paths and then direct each message to the proper handler.
//
// Minimal interface modeled after net/http.ServeMux
//
// TODO: look for better trie routers that handle patterns...
// maybe take code from here?
// https://github.com/julienschmidt/httprouter
// https://github.com/julienschmidt/httprouter/blob/master/tree.go
type Router struct {
	routes map[string]weave.Handler
}

var _ weave.Registry = (*Router)(nil)
var _ weave.Handler = (*Router)(nil)

// NewRouter returns a new empty router instance.
func NewRouter() *Router {
	return &Router{
		routes: make(map[string]weave.Handler),
	}
}

// Handle implements weave.Registry interface.
func (r *Router) Handle(m weave.Msg, h weave.Handler) {
	path := m.Path()
	if !isPath(path) {
		panic(fmt.Sprintf("invalid path: %T: %s", m, path))
	}
	if _, ok := r.routes[path]; ok {
		panic(fmt.Sprintf("re-registering route: %T: %s", m, path))
	}
	r.routes[path] = h
}

// handler returns the registered Handler for this path. If no path is found,
// returns a noSuchPath Handler.  This method always returns a non-nil Handler.
func (r *Router) handler(m weave.Msg) weave.Handler {
	path := m.Path()
	if h, ok := r.routes[path]; ok {
		return h
	}
	return notFoundHandler(path)
}

// Check dispatches to the proper handler based on path
func (r *Router) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	msg, err := tx.GetMsg()
	if err != nil {
		return nil, errors.Wrap(err, "cannot load msg")
	}
	h := r.handler(msg)
	return h.Check(ctx, store, tx)
}

// Deliver dispatches to the proper handler based on path
func (r *Router) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := tx.GetMsg()
	if err != nil {
		return nil, errors.Wrap(err, "cannot load msg")
	}
	h := r.handler(msg)
	return h.Deliver(ctx, store, tx)
}

// notFoundHandler always returns ErrNotFound error regardless of the arguments
// provided.
type notFoundHandler string

func (path notFoundHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	return nil, errors.Wrapf(errors.ErrNotFound, "no handler for message path %q", path)
}

func (path notFoundHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	return nil, errors.Wrapf(errors.ErrNotFound, "no handler for message path %q", path)
}
