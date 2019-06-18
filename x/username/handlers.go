package username

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

const (
	registerUsernameTokenCost     = 0
	transferUsernameTokenCost     = 0
	changeUsernameTokenTargetCost = 0
)

func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	r = migration.SchemaMigratingRegistry("username", r)

	b := NewUsernameTokenBucket()
	r.Handle(RegisterUsernameTokenMsg{}.Path(), &registerUsernameTokenHandler{auth: auth, bucket: b})
	r.Handle(TransferUsernameTokenMsg{}.Path(), &transferUsernameTokenHandler{auth: auth, bucket: b})
	r.Handle(ChangeUsernameTokenTargetsMsg{}.Path(), &changeUsernameTokenTargetsHandler{auth: auth, bucket: b})
}

type registerUsernameTokenHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
}

func (h *registerUsernameTokenHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: registerUsernameTokenCost}, nil
}

func (h *registerUsernameTokenHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	owner := x.MainSigner(ctx, h.auth).Address()
	if len(owner) == 0 {
		return nil, errors.Wrap(errors.ErrUnauthorized, "message must be signed")
	}

	token := UsernameToken{
		Metadata: &weave.Metadata{Schema: 1},
		Targets:  msg.Targets,
		Owner:    owner,
	}
	if _, err := h.bucket.Put(db, msg.Username.Bytes(), &token); err != nil {
		return nil, errors.Wrap(err, "cannot store token")
	}
	return &weave.DeliverResult{Data: msg.Username.Bytes()}, nil
}

func (h *registerUsernameTokenHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*RegisterUsernameTokenMsg, error) {
	var msg RegisterUsernameTokenMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	switch err := h.bucket.Has(db, msg.Username.Bytes()); {
	case err == nil:
		return nil, errors.Wrapf(errors.ErrDuplicate, "username %q already registered", msg.Username)
	case errors.ErrNotFound.Is(err):
		// All good. Username is not taken yet.
	default:
		return nil, errors.Wrap(err, "cannot check if username is unique")
	}
	return &msg, nil
}

type transferUsernameTokenHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
}

func (h *transferUsernameTokenHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: transferUsernameTokenCost}, nil
}

func (h *transferUsernameTokenHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, token, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	token.Owner = msg.NewOwner
	if _, err := h.bucket.Put(db, msg.Username.Bytes(), token); err != nil {
		return nil, errors.Wrap(err, "cannot store token")
	}
	return &weave.DeliverResult{Data: msg.Username.Bytes()}, nil
}

func (h *transferUsernameTokenHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*TransferUsernameTokenMsg, *UsernameToken, error) {
	var msg TransferUsernameTokenMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}

	var token UsernameToken
	if err := h.bucket.One(db, msg.Username.Bytes(), &token); err != nil {
		return nil, nil, errors.Wrap(err, "cannot get token from database")
	}

	if !h.auth.HasAddress(ctx, token.Owner) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "only the token owner can execute this operation")
	}

	return &msg, &token, nil
}

type changeUsernameTokenTargetsHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
}

func (h *changeUsernameTokenTargetsHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: changeUsernameTokenTargetCost}, nil
}

func (h *changeUsernameTokenTargetsHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, token, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	token.Targets = msg.NewTargets
	if _, err := h.bucket.Put(db, msg.Username.Bytes(), token); err != nil {
		return nil, errors.Wrap(err, "cannot store token")
	}
	return &weave.DeliverResult{Data: msg.Username.Bytes()}, nil
}

func (h *changeUsernameTokenTargetsHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*ChangeUsernameTokenTargetsMsg, *UsernameToken, error) {
	var msg ChangeUsernameTokenTargetsMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}

	var token UsernameToken
	if err := h.bucket.One(db, msg.Username.Bytes(), &token); err != nil {
		return nil, nil, errors.Wrap(err, "cannot get token from database")
	}

	if !h.auth.HasAddress(ctx, token.Owner) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "only the token owner can execute this operation")
	}

	return &msg, &token, nil
}
