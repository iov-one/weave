package migration

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x"
)

// SchemaMigratingRegistry decorates given registry to always migrate schema of
// an incoming message, before passing it down to a registered handler.
// Decorating a registry with this function is equivalent to using a raw
// registry and wrapping each handler with SchemaMigratingHandler before
// registering.
// This function force all registration to use schema migration and must not
// be used together with SchemaRoutingHandler functionality.
func SchemaMigratingRegistry(packageName string, r weave.Registry) weave.Registry {
	return &schemaMigratingRegistry{
		packageName: packageName,
		reg:         r,
	}
}

type schemaMigratingRegistry struct {
	packageName string
	reg         weave.Registry
}

func (r *schemaMigratingRegistry) Handle(m weave.Msg, h weave.Handler) {
	r.reg.Handle(m, SchemaMigratingHandler(r.packageName, h))
}

// SchemaMigratingHandler returns a weave handler that will ensure incoming
// messages are in the current schema version format. If a message in older
// schema is handled then it is first being migrated. Messages that cannot be
// migrated to current schema version are returning migration error. This
// functionality is executed before the decorated handler and it is completely
// transparent to the wrapped handler.
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
		return errors.Wrap(errors.ErrMsg, "message cannot be migrated")
	}
	currSchemaVer, err := h.schema.CurrentSchema(db, h.packageName)
	if err != nil {
		return errors.Wrap(err, "current message schema")
	}

	// Migration is applied in place, directly modifying the instance.
	if err := h.migrations.Apply(db, m, currSchemaVer); err != nil {
		return errors.Wrap(err, "schema migration")
	}
	return nil
}

// RegisterRoutes registers handlers for feedlist message processing.
func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	bucket := NewSchemaBucket()
	r.Handle(&UpgradeSchemaMsg{}, &upgradeSchemaHandler{
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

// SchemaRoutingHandler clubs together message handlers for a single type
// message but different schema formats. Each handler is registered together
// with the lowest schema version that it supports. For example
//
//   handler := SchemaRoutingHandler([]weave.Handler{
//     1: &MyHandlerVersionAlpha{},
//     7: &MyHandlerVersionBeta{},
//   })
//
// In the above setup, messages with schema version 1 to 6 will be handled by
// the alpha handler. Messages with schema version 7 and above are passed to
// the beta handler.
//
// It is not allowed to use an empty schemaRoutingHandler instance. It is not
// allowed to register a handler for schema version zero. This function panics
// if any of those requirements is not met.
//
// All messages processed by this handler must implement Migratable interface.
func SchemaRoutingHandler(handlers []weave.Handler) weave.Handler {
	if len(handlers) == 0 {
		err := errors.Wrap(errors.ErrHuman, "no handler registered")
		panic(err)
	}
	if handlers[0] != nil {
		err := errors.Wrap(errors.ErrHuman, "zero schema version handler must not be registered")
		panic(err)
	}
	return schemaRoutingHandler(handlers)
}

type schemaRoutingHandler []weave.Handler

var _ weave.Handler = (schemaRoutingHandler)(nil)

func (h schemaRoutingHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	handler, err := h.selectHandler(tx)
	if err != nil {
		return nil, err
	}
	return handler.Check(ctx, db, tx)
}

func (h schemaRoutingHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	handler, err := h.selectHandler(tx)
	if err != nil {
		return nil, err
	}
	return handler.Deliver(ctx, db, tx)
}

// selectHandler returns the best fitting handler to process given transaction,
// selected by introspecting the transaction message schema version.
func (h schemaRoutingHandler) selectHandler(tx weave.Tx) (weave.Handler, error) {
	msg, err := tx.GetMsg()
	if err != nil {
		return nil, errors.Wrap(err, "cannot get transaction message")
	}
	m, ok := msg.(Migratable)
	if !ok {
		return nil, errors.Wrapf(errors.ErrType, "message %T does not support schema versioning", msg)
	}
	meta := m.GetMetadata()

	for ver := meta.Schema; ver > 0; ver-- {
		if h[ver] != nil {
			return h[ver], nil
		}
	}
	return nil, errors.Wrapf(errors.ErrSchema, "no matching handler for schema version %d", meta.Schema)
}
