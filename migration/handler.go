package migration

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x"
)

// SchemaMigratingHandler returns a weave handler that will ensure incomming
// messages are in the curren schema version format. If a message in older
// schema is handled then it is first being migrated. Messages that cannot be
// migrated to current schema version are returning migration error. This
// functionality is executed before the decorated handler and it is completely
// transpared to the wrapped handler.
func SchemaMigratingHandler(packageName string, h weave.Handler) weave.Handler {
	return &schemaMigratingHandler{
		handler:     h,
		packageName: packageName,
		schema:      NewSchemaBucket(),
		migrations:  reg,
	}
}

type schemaMigratingHandler struct {
	handler     weave.Handler
	packageName string
	schema      *SchemaBucket
	migrations  *register
}

func (h *schemaMigratingHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if err := h.migrate(db, tx); err != nil {
		return nil, errors.Wrap(err, "migration")
	}
	return h.handler.Check(ctx, db, tx)
}

func (h *schemaMigratingHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	if err := h.migrate(db, tx); err != nil {
		return nil, errors.Wrap(err, "migration")
	}
	return h.handler.Deliver(ctx, db, tx)
}

func (h *schemaMigratingHandler) migrate(db weave.ReadOnlyKVStore, tx weave.Tx) error {
	msg, err := tx.GetMsg()
	if err != nil {
		return errors.Wrap(err, "get msg")
	}

	m, ok := msg.(Migratable)
	if !ok {
		return errors.Wrap(errors.ErrInvalidMsg, "message cannot be migrated")
	}
	currSchemaVer, err := h.schema.CurrentSchema(db, h.packageName)
	if err != nil {
		return errors.Wrap(err, "current message schema")
	}

	// Migration is applied in place, directly modyfying the instance.
	if err := h.migrations.Apply(db, m, currSchemaVer); err != nil {
		return errors.Wrap(err, "schema migration")
	}
	return nil
}

// RegisterRoutes registers handlers for feedlist message processing.
func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	bucket := NewSchemaBucket()
	r.Handle(pathUpgradeSchemaMsg, &upgradeSchemaHandler{
		bucket: bucket,
		auth:   auth,
	})
}

type upgradeSchemaHandler struct {
	bucket *SchemaBucket
	auth   x.Authenticator
}

func (h *upgradeSchemaHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{}, nil
}

func (h *upgradeSchemaHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	ver, err := h.bucket.CurrentSchema(db, msg.Pkg)
	if err != nil && !errors.ErrNotFound.Is(err) {
		return nil, errors.Wrap(err, "current schema version")
	}

	schema := Schema{
		Metadata: &weave.Metadata{Schema: 1},
		Pkg:      msg.Pkg,
		Version:  ver + 1,
	}
	obj, err := h.bucket.Create(db, &schema)
	if err != nil {
		return nil, errors.Wrap(err, "create schema version")
	}

	return &weave.DeliverResult{Data: obj.Key()}, nil
}

func (h *upgradeSchemaHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*UpgradeSchemaMsg, error) {
	var msg UpgradeSchemaMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	conf := mustLoadConf(db)
	if !h.auth.HasAddress(ctx, conf.Admin) {
		return nil, errors.Wrap(errors.ErrUnauthorized, "admin signature required")
	}

	return &msg, nil
}

// SchemaRoutingHandler is a container that clubs toghether message handlers
// for a single type message but different schema formats. Each handler is
// registered together with the lowest schema version that it supports. For example
//
//   handler := SchemaRoutingHandler{
//     1: &MyHandlerVersionAlpha{},
//     7: &MyHandlerVersionBeta{},
//   }
//
// In the above setup, messages with schema version 1 to 6 will be handled by
// the alpha handler. Messages with schema version 7 and above are passed to
// the beta handler.
//
// It is not allowed to use an empty SchemaRoutingHandler instance. It is not
// allowed to register a handler for schema version zero.
// All messages processed by this handler must implement Migratable interface.
type SchemaRoutingHandler []weave.Handler

var _ weave.Handler = (SchemaRoutingHandler)(nil)

func (h SchemaRoutingHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	handler, err := h.selectHandler(tx)
	if err != nil {
		return nil, err
	}
	return handler.Check(ctx, db, tx)
}

func (h SchemaRoutingHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	handler, err := h.selectHandler(tx)
	if err != nil {
		return nil, err
	}
	return handler.Deliver(ctx, db, tx)
}

// selectHandler returns the best fitting handler to process given transaction,
// selected by introspecting the transaction message schema version.
func (h SchemaRoutingHandler) selectHandler(tx weave.Tx) (weave.Handler, error) {
	if len(h) == 0 {
		return nil, errors.Wrap(errors.ErrHuman, "no handler registered")
	}
	if h[0] != nil {
		return nil, errors.Wrap(errors.ErrHuman, "zero schema version handler must not be registered")
	}

	msg, err := tx.GetMsg()
	if err != nil {
		return nil, errors.Wrap(err, "cannot get transaction message")
	}
	m, ok := msg.(Migratable)
	if !ok {
		return nil, errors.Wrapf(errors.ErrInvalidType, "message %T does not support schema versioning", msg)
	}
	meta := m.GetMetadata()

	var handler weave.Handler
	for ver := uint32(1); ver < uint32(len(h)); ver++ {
		// It is allowed to leave gaps between handler version mappings
		// so it. If this is the case, the previously available version
		// must be used.
		if next := h[ver]; next != nil {
			handler = next
		}
		if ver >= meta.Schema {
			break
		}
	}
	if handler == nil {
		return nil, errors.Wrapf(errors.ErrSchema, "no matching handler for schema version %d", meta.Schema)
	}
	return handler, nil
}
