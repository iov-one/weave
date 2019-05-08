package validators

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/x"
	abci "github.com/tendermint/tendermint/abci/types"
)

// RegisterRoutes will instantiate and register
// all handlers in this package.
func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	bucket := NewAccountBucket()
	r.Handle(pathUpdate, migration.SchemaMigratingHandler("validators", &updateHandler{
		auth:   auth,
		bucket: bucket,
	}))
}

// RegisterQuery will register this bucket as "/validators".
func RegisterQuery(qr weave.QueryRouter) {
	NewAccountBucket().Register("validators", qr)
}

type updateHandler struct {
	auth   x.Authenticator
	bucket *AccountBucket
}

var _ weave.Handler = (*updateHandler)(nil)

func (h updateHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, store, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{}, nil
}

func (h updateHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	diff, err := h.validate(ctx, store, tx)
	if err != nil {
		return nil, err
	}
	return &weave.DeliverResult{Diff: diff}, nil
}

func (h updateHandler) validate(ctx weave.Context, store weave.KVStore, tx weave.Tx) ([]abci.ValidatorUpdate, error) {
	var msg SetValidatorsMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	diff := msg.AsABCI()
	if len(diff) == 0 {
		return nil, errors.Wrap(errors.ErrEmpty, "diff")
	}

	accounts, err := h.bucket.GetAccounts(store)
	if err != nil {
		return nil, err
	}

	var hasPermission bool
	for _, addr := range accounts.Addresses {
		if h.auth.HasAddress(ctx, addr) {
			hasPermission = true
			break
		}
	}
	if !hasPermission {
		return nil, errors.Wrap(errors.ErrUnauthorized, "no permission")
	}

	return diff, nil
}
