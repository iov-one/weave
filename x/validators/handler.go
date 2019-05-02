package validators

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/x"
	abci "github.com/tendermint/tendermint/abci/types"
)

type AuthCheckAddress = func(auth x.Authenticator, ctx weave.Context) CheckAddress

var authCheckAddress = func(auth x.Authenticator, ctx weave.Context) CheckAddress {
	return func(addr weave.Address) bool {
		return auth.HasAddress(ctx, addr)
	}
}

// RegisterRoutes will instantiate and register
// all handlers in this package.
func RegisterRoutes(r weave.Registry, auth x.Authenticator, control Controller) {
	r.Handle(pathUpdate, migration.SchemaMigratingHandler("validators", NewUpdateHandler(auth, control, authCheckAddress)))
}

// RegisterQuery will register this bucket as "/validators".
func RegisterQuery(qr weave.QueryRouter) {
	NewBucket().Register("validators", qr)
}

// UpdateHandler will handle sending coins.
type UpdateHandler struct {
	auth             x.Authenticator
	control          Controller
	authCheckAddress AuthCheckAddress
}

var _ weave.Handler = UpdateHandler{}

// NewUpdateHandler creates a handler for SendMsg.
func NewUpdateHandler(auth x.Authenticator, control Controller, checkAddr AuthCheckAddress) UpdateHandler {
	return UpdateHandler{
		auth:             auth,
		control:          control,
		authCheckAddress: checkAddr,
	}
}

// Check verifies all the preconditions.
func (h UpdateHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, store, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{}, nil
}

// Deliver provides the diff given everything is okay with permissions and such
// Check did the same job already, so we can assume stuff goes okay.
func (h UpdateHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	diff, err := h.validate(ctx, store, tx)
	if err != nil {
		return nil, err
	}
	return &weave.DeliverResult{Diff: diff}, nil
}

func (h UpdateHandler) validate(ctx weave.Context, store weave.KVStore, tx weave.Tx) ([]abci.ValidatorUpdate, error) {
	var msg SetValidatorsMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}
	return h.control.CanUpdateValidators(store, h.authCheckAddress(h.auth, ctx), msg.AsABCI())
}
