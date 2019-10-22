package lateinit

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

// RegisterRoutes registers handlers for message processing.
func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	r = migration.SchemaMigratingRegistry("lateinit", r)
	bucket := NewExecutedInitBucket()
	r.Handle(&ExecuteInitMsg{}, &executeInitHandler{
		auth:   auth,
		bucket: bucket,
		reg:    reg,
	})
}

type executeInitHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
	reg    *register
}

func (h *executeInitHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, errors.Wrap(err, "invalid message")
	}
	if err := h.reg.Exec(ctx, h.auth, db, msg.InitID); err != nil {
		return nil, errors.Wrap(err, "cannot initialize entity")
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *executeInitHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, errors.Wrap(err, "invalid message")
	}

	if err := h.reg.Exec(ctx, h.auth, db, msg.InitID); err != nil {
		return nil, errors.Wrap(err, "cannot initialize entity")
	}

	// Create a record in the database to remember that this initialization
	// was executed.
	fix := &ExecutedInit{
		Metadata: &weave.Metadata{},
		InitID:   msg.InitID,
	}
	if h.bucket.Put(db, []byte(msg.InitID), fix); err != nil {
		return nil, errors.Wrap(err, "cannot persist information about executed fix")
	}

	return &weave.DeliverResult{}, nil
}

func (h *executeInitHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*ExecuteInitMsg, error) {
	var msg ExecuteInitMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	switch sig, err := h.reg.RequiredSigner(msg.InitID); {
	case err == nil:
		// Transaction must be signed only if the signature is
		// required.
		if len(sig) != 0 && !h.auth.HasAddress(ctx, sig) {
			return nil, errors.ErrUnauthorized
		}
	case errors.ErrNotFound.Is(err):
		return nil, errors.Wrap(err, "no initializator")
	default:
		return nil, errors.Wrap(err, "cannot get initializator signer")
	}

	switch err := h.bucket.Has(db, []byte(msg.InitID)); {
	case errors.ErrNotFound.Is(err):
		// All good.
	case err == nil:
		return nil, errors.Wrap(errors.ErrState, "entity already initialized")
	default:
		return nil, errors.Wrap(err, "cannot check if entity is already initialized")
	}
	return &msg, nil
}

func RegisterQuery(qr weave.QueryRouter) {
	NewExecutedInitBucket().Register("executedfixes", qr)
}
