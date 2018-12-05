package token

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

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

const (
	newTokenInfoCost = 100
)

func (h *TokenInfoHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}
	obj := orm.NewSimpleObj([]byte(msg.Ticker), &TokenInfo{
		Name:    msg.Name,
		SigFigs: msg.SigFigs,
	})
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
		return nil, errors.ErrUnauthorized()
	}

	// Token can be registered only once and must not be updated.
	if obj, err := h.bucket.Get(db, msg.Ticker); err != nil {
		return nil, err
	} else if obj != nil {
		return nil, ErrDuplicateToken(msg.Ticker)
	}

	return msg, nil
}
