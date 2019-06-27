package username

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

const (
	registerTokenCost     = 0
	transferTokenCost     = 0
	changeTokenTargetCost = 0
)

func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	r = migration.SchemaMigratingRegistry("username", r)

	b := NewTokenBucket()
	r.Handle(&RegisterTokenMsg{}, &registerTokenHandler{auth: auth, bucket: b})
	r.Handle(&TransferTokenMsg{}, &transferTokenHandler{auth: auth, bucket: b})
	r.Handle(&ChangeTokenTargetsMsg{}, &changeTokenTargetsHandler{auth: auth, bucket: b})
}

type registerTokenHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
}

func (h *registerTokenHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: registerTokenCost}, nil
}

func (h *registerTokenHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	owner := x.MainSigner(ctx, h.auth).Address()
	if len(owner) == 0 {
		return nil, errors.Wrap(errors.ErrUnauthorized, "message must be signed")
	}

	token := Token{
		Metadata: &weave.Metadata{Schema: 1},
		Targets:  msg.Targets,
		Owner:    owner,
	}
	if _, err := h.bucket.Put(db, msg.Username.Bytes(), &token); err != nil {
		return nil, errors.Wrap(err, "cannot store token")
	}
	return &weave.DeliverResult{Data: msg.Username.Bytes()}, nil
}

func (h *registerTokenHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*RegisterTokenMsg, error) {
	var msg RegisterTokenMsg
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

type transferTokenHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
}

func (h *transferTokenHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: transferTokenCost}, nil
}

func (h *transferTokenHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
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

func (h *transferTokenHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*TransferTokenMsg, *Token, error) {
	var msg TransferTokenMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}

	var token Token
	if err := h.bucket.One(db, msg.Username.Bytes(), &token); err != nil {
		return nil, nil, errors.Wrap(err, "cannot get token from database")
	}

	if !h.auth.HasAddress(ctx, token.Owner) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "only the token owner can execute this operation")
	}

	return &msg, &token, nil
}

type changeTokenTargetsHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
}

func (h *changeTokenTargetsHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: changeTokenTargetCost}, nil
}

func (h *changeTokenTargetsHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
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

func (h *changeTokenTargetsHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*ChangeTokenTargetsMsg, *Token, error) {
	var msg ChangeTokenTargetsMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}

	if err := msg.Username.Validate(); err != nil {
		return nil, nil, errors.Wrap(err, "username")
	}

	var token Token
	if err := h.bucket.One(db, msg.Username.Bytes(), &token); err != nil {
		return nil, nil, errors.Wrap(err, "cannot get token from database")
	}

	if !h.auth.HasAddress(ctx, token.Owner) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "only the token owner can execute this operation")
	}

	return &msg, &token, nil
}
