package app

import (
	"fmt"
	"regexp"

	"github.com/confio/weave"
	"github.com/confio/weave/errors"
)

// DefaultRouterSize preallocates this much space to hold routes
const DefaultRouterSize = 10

// isPath is the RegExp to ensure the routes make sense
var isPath = regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString

// CodeNoSuchPath is an ABCI Response Codes
// Base SDK reserves 0 ~ 99. App uses 10 ~ 19
const CodeNoSuchPath uint32 = 10

var errNoSuchPath = fmt.Errorf("Path not registered")

// ErrNoSuchPath constructs an error when router doesn't know the path
func ErrNoSuchPath(path string) error {
	return errors.WithLog(path, errNoSuchPath, CodeNoSuchPath)
}

// IsNoSuchPathErr checks if this is an unknown route error
func IsNoSuchPathErr(err error) bool {
	return errors.IsSameError(errNoSuchPath, err)
}

// Router allows us to register many handlers with different
// paths and then direct each message to the proper handler.
//
// Minimal interface modeled after net/http.ServeMux
//
// TODO: look for better trie routers that handle patterns...
type Router struct {
	routes map[string]weave.Handler
}

var _ weave.Registry = Router{}
var _ weave.Handler = Router{}

// NewRouter initializes a router with no routes
func NewRouter() Router {
	return Router{
		routes: make(map[string]weave.Handler, DefaultRouterSize),
	}
}

// Handle adds a new Handler for the given path.
// panics if another Handler was already registered
func (r Router) Handle(path string, h weave.Handler) {
	if !isPath(path) {
		panic(fmt.Sprintf("Invalid path: %s", path))
	}
	if _, ok := r.routes[path]; ok {
		panic(fmt.Sprintf("Re-registering route: %s", path))
	}
	r.routes[path] = h
}

// Handler returns the registered Handler for this path.
// If no path is found, returns a noSuchPath Handler
// Always returns a non-nil Handler
func (r Router) Handler(path string) weave.Handler {
	h, ok := r.routes[path]
	if !ok {
		return noSuchPathHandler{path}
	}
	return h
}

// Check dispatches to the proper handler based on path
func (r Router) Check(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.CheckResult, error) {

	path := tx.GetMsg().Path()
	h := r.Handler(path)
	return h.Check(ctx, store, tx)
}

// Deliver dispatches to the proper handler based on path
func (r Router) Deliver(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.DeliverResult, error) {

	path := tx.GetMsg().Path()
	h := r.Handler(path)
	return h.Deliver(ctx, store, tx)
}

//-------------------- error handler ---------------

type noSuchPathHandler struct {
	path string
}

var _ weave.Handler = noSuchPathHandler{}

// Check always returns ErrNoSuchPath
func (h noSuchPathHandler) Check(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.CheckResult, error) {

	return weave.CheckResult{}, ErrNoSuchPath(h.path)
}

// Deliver always returns ErrNoSuchPath
func (h noSuchPathHandler) Deliver(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.DeliverResult, error) {

	return weave.DeliverResult{}, ErrNoSuchPath(h.path)
}
