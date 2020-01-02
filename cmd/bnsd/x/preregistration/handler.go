package preregistration

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	r = migration.SchemaMigratingRegistry("preregistration", r)

	records := NewRecordBucket()
	r.Handle(&RegisterMsg{}, &registerHandler{
		records: records,
		auth:    auth,
	})
}

type registerHandler struct {
	auth    x.Authenticator
	records orm.ModelBucket
}

func (h *registerHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *registerHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	rec := &Record{
		Metadata: &weave.Metadata{Schema: 1},
		Domain:   msg.Domain,
		Owner:    msg.Owner,
	}
	if _, err := h.records.Put(db, []byte(msg.Domain), rec); err != nil {
		return nil, errors.Wrap(err, "save record")
	}
	return &weave.DeliverResult{Data: nil}, nil
}

func (h *registerHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*RegisterMsg, error) {
	var msg RegisterMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	conf, err := loadConf(db)
	if err != nil {
		return nil, errors.Wrap(err, "load configuration")
	}

	if !h.auth.HasAddress(ctx, conf.Owner) {
		return nil, errors.Wrap(errors.ErrUnauthorized, "configuration owner signature is required")
	}
	switch err := h.records.Has(db, []byte(msg.Domain)); {
	case err == nil:
		return nil, errors.Wrap(errors.ErrDuplicate, "domain already registered")
	case errors.ErrNotFound.Is(err):
		return &msg, nil
	default:
		return nil, errors.Wrap(err, "has domain")
	}
}
