package datamigration

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

// RegisterRoutes registers handlers for message processing.
func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	r = migration.SchemaMigratingRegistry("datamigration", r)
	bucket := NewExecutedMigrationBucket()
	r.Handle(&ExecuteMigrationMsg{}, &executeMigrationHandler{
		auth:   auth,
		bucket: bucket,
		reg:    reg,
	})
}

type executeMigrationHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
	reg    *register
}

func (h *executeMigrationHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, errors.Wrap(err, "invalid message")
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *executeMigrationHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, mig, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, errors.Wrap(err, "invalid message")
	}

	if err := mig.Migrate(ctx, db); err != nil {
		return nil, errors.Wrap(err, "migration failed")
	}

	// Create a record in the database to remember that this migration was
	// executed.
	fix := &ExecutedMigration{
		Metadata: &weave.Metadata{},
	}
	if _, err := h.bucket.Put(db, []byte(msg.MigrationID), fix); err != nil {
		return nil, errors.Wrap(err, "cannot persist an information about executed migration")
	}

	return &weave.DeliverResult{}, nil
}

func (h *executeMigrationHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*ExecuteMigrationMsg, *Migration, error) {
	var msg ExecuteMigrationMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}

	m, err := h.reg.Migration(msg.MigrationID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot get migration")
	}

	for _, s := range m.RequiredSigners {
		if !h.auth.HasAddress(ctx, s) {
			return nil, nil, errors.Wrap(errors.ErrUnauthorized, "missing signature")
		}
	}

	if m.ChainID != weave.GetChainID(ctx) {
		return nil, nil, errors.Wrapf(errors.ErrChain, "allowed only on %q chain", m.ChainID)
	}

	switch err := h.bucket.Has(db, []byte(msg.MigrationID)); {
	case errors.ErrNotFound.Is(err):
		// All good.
	case err == nil:
		return nil, nil, errors.Wrap(errors.ErrState, "migration already executed")
	default:
		return nil, nil, errors.Wrap(err, "cannot check if migration was executed")
	}
	return &msg, m, nil
}

func RegisterQuery(qr weave.QueryRouter) {
	NewExecutedMigrationBucket().Register("executedmigrations", qr)
}
