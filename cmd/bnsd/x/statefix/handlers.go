package statefix

import (
	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

// RegisterRoutes registers handlers for message processing.
func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	r = migration.SchemaMigratingRegistry("statefix", r)
	bucket := NewExecutedFixBucket()
	r.Handle(&ExecuteFixMsg{}, &executeFixHandler{
		auth:   auth,
		bucket: bucket,
	})
}

type executeFixHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
}

func (h *executeFixHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, errors.Wrap(err, "invalid message")
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *executeFixHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, fixFn, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, errors.Wrap(err, "invalid message")
	}

	if err := fixFn(ctx, db); err != nil {
		return nil, errors.Wrap(err, "fix function failed")
	}

	// Create a record in the database to remember that this fix was executed.
	fix := &ExecutedFix{
		Metadata: &weave.Metadata{},
		FixID:    msg.FixID,
	}
	if h.bucket.Put(db, []byte(msg.FixID), fix); err != nil {
		return nil, errors.Wrap(err, "cannot persist information about executed fix")
	}

	return &weave.DeliverResult{}, nil
}

func (h *executeFixHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*ExecuteFixMsg, FixFunc, error) {
	//
	// TODO XXX
	//
	// If possible and makes sense, require signature of the whole
	// governing board.
	//
	// if !h.auth.HasAddress(ctx, govSig) {
	// 	return nil, nil, errors.Wrap(errors.ErrUnauthorized, "government signature must be present")
	// }

	var msg ExecuteFixMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}

	fn, ok := fixes[msg.FixID]
	if !ok {
		return nil, nil, errors.Wrapf(errors.ErrNotFound, "fix function %q not defined", msg.FixID)
	}

	switch err := h.bucket.Has(db, []byte(msg.FixID)); {
	case err == nil:
		return nil, nil, errors.Wrapf(errors.ErrDuplicate, "fix %q already executed", msg.FixID)
	case errors.ErrNotFound.Is(err):
		// All good.
	default:
		return nil, nil, errors.Wrap(err, "cannot check if fix was executed")
	}

	return &msg, fn, nil
}

func RegisterQuery(qr weave.QueryRouter) {
	NewExecutedFixBucket().Register("executedfixes", qr)
}
