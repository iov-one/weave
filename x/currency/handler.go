package currency

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x"
)

const newTokenInfoCost = 100

func RegisterQuery(qr weave.QueryRouter) {
	NewTokenInfoBucket().Register("tokens", qr)
}

func RegisterRoutes(r weave.Registry, auth x.Authenticator, issuer weave.Address) {
	r.Handle(NewTokenInfoMsg{}.Path(), NewTokenInfoHandler(auth, issuer))
}

func NewTokenInfoHandler(auth x.Authenticator, issuer weave.Address) weave.Handler {
	return &TokenInfoHandler{
		auth:   auth,
		issuer: issuer,
		bucket: NewTokenInfoBucket(),
	}
}

type TokenInfoHandler struct {
	auth   x.Authenticator
	bucket *TokenInfoBucket
	issuer weave.Address
}

func (h *TokenInfoHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	if _, err := h.validate(ctx, db, tx); err != nil {
		return res, err
	}
	res.GasAllocated += newTokenInfoCost
	return res, nil
}

func (h *TokenInfoHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}
	obj := NewTokenInfo(msg.Ticker, msg.Name, msg.SigFigs)
	return res, h.bucket.Save(db, obj)
}

func (h *TokenInfoHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*NewTokenInfoMsg, error) {
	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}
	msg, ok := rmsg.(*NewTokenInfoMsg)
	if !ok {
		return nil, errors.ErrUnknownTxType(rmsg)
	}

	if err := msg.Validate(); err != nil {
		return nil, err
	}

	// Ensure we have permission if the issuer is provided.
	if h.issuer != nil && !h.auth.HasAddress(ctx, h.issuer) {
		return nil, errors.ErrUnauthorized
	}

	// Token can be registered only once and must not be updated.
	switch obj, err := h.bucket.Get(db, msg.Ticker); {
	case err != nil:
		return nil, err
	case obj != nil:
		return nil, errors.ErrDuplicate.Newf("ticker %s", msg.Ticker)
	}

	return msg, nil
}
