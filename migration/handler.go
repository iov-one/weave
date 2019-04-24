package migration

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x"
)

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

	schema := Schema{Pkg: msg.Pkg, Version: ver + 1}
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
