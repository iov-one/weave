package validators

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/x"
)

// RegisterRoutes will instantiate and register
// all handlers in this package.
func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	bucket := NewAccountBucket()
	r.Handle(&ApplyDiffMsg{}, migration.SchemaMigratingHandler("validators", &updateHandler{
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
	if _, _, err := h.validate(ctx, store, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{}, nil
}

func (h updateHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	diff, updates, err := h.validate(ctx, store, tx)
	if err != nil {
		return nil, err
	}
	err = weave.StoreValidatorUpdates(store, updates)
	if err != nil {
		return nil, errors.Wrap(err, "store validator updates")
	}

	return &weave.DeliverResult{Diff: diff}, nil
}

// Validate returns an update diff, ValidatorUpdates to store for bookkeeping and an error.
func (h updateHandler) validate(ctx weave.Context, store weave.KVStore, tx weave.Tx) ([]weave.ValidatorUpdate,
	weave.ValidatorUpdates, error) {
	var msg ApplyDiffMsg
	var resUpdates weave.ValidatorUpdates
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, resUpdates, errors.Wrap(err, "load msg")
	}

	diff := msg.ValidatorUpdates
	if len(diff) == 0 {
		return nil, resUpdates, errors.Wrap(errors.ErrEmpty, "diff")
	}

	accounts, err := h.bucket.GetAccounts(store)
	if err != nil {
		return nil, resUpdates, err
	}

	var hasPermission bool
	for _, addr := range accounts.Addresses {
		if h.auth.HasAddress(ctx, addr) {
			hasPermission = true
			break
		}
	}
	if !hasPermission {
		return nil, resUpdates, errors.Wrap(errors.ErrUnauthorized, "no permission")
	}

	updates, err := weave.GetValidatorUpdates(store)
	if err != nil {
		return nil, resUpdates, errors.Wrap(err, "failed to query validators")
	}

	resUpdates = updates

	for _, v := range diff {
		if validator, key, ok := resUpdates.Get(v.PubKey); ok {
			if v.Power == validator.Power {
				return nil, resUpdates, errors.Wrap(errors.ErrInput, "same validator power")
			}
			resUpdates.ValidatorUpdates[key] = v
			continue
		}

		if v.Power == 0 {
			return nil, resUpdates, errors.Wrap(errors.ErrInput, "setting unknown validator power to 0")
		}

		resUpdates.ValidatorUpdates = append(resUpdates.ValidatorUpdates, v)
	}

	// Deduplicate updates for storage.
	return diff, resUpdates.Deduplicate(true), nil
}
