package currency

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/x"
)

const newTokenInfoCost = 100

func RegisterQuery(qr weave.QueryRouter) {
	NewTokenInfoBucket().Register("tokens", qr)
}

func RegisterRoutes(r weave.Registry, auth x.Authenticator, issuer weave.Address) {
	r = migration.SchemaMigratingRegistry("currency", r)

	r.Handle(&CreateMsg{}, newCreateTokenInfoHandler(auth, issuer))
}

func newCreateTokenInfoHandler(auth x.Authenticator, issuer weave.Address) weave.Handler {
	return &createTokenInfoHandler{
		auth:   auth,
		issuer: issuer,
		bucket: NewTokenInfoBucket(),
	}
}

type createTokenInfoHandler struct {
	auth   x.Authenticator
	bucket *TokenInfoBucket
	issuer weave.Address
}

func (h *createTokenInfoHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: newTokenInfoCost}, nil
}

func (h *createTokenInfoHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	obj := NewTokenInfo(msg.Ticker, msg.Name)
	return &weave.DeliverResult{}, h.bucket.Save(db, obj)
}

func (h *createTokenInfoHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*CreateMsg, error) {
	var msg CreateMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	// Ensure we have permission if the issuer is provided.
	if h.issuer != nil && !h.auth.HasAddress(ctx, h.issuer) {
		return nil, errors.Wrapf(errors.ErrUnauthorized, "Token only issued by %s", h.issuer)
	}

	// Token can be registered only once and must not be updated.
	switch obj, err := h.bucket.Get(db, msg.Ticker); {
	case err != nil:
		return nil, err
	case obj != nil:
		return nil, errors.Wrapf(errors.ErrDuplicate, "ticker %s", msg.Ticker)
	}

	return &msg, nil
}
