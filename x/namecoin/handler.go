package namecoin

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
)

// NewFeeDecorator customizes cash/FeeDecorator to use our
// WalletBucket.
func NewFeeDecorator(auth x.Authenticator) weave.Decorator {
	return cash.NewFeeDecorator(auth, NewController())
}

// NewSendHandler customizes cash/SendHandler to use our
// WalletBucket.
func NewSendHandler(auth x.Authenticator) weave.Handler {
	return cash.NewSendHandler(auth, NewController())
}

// NewTokenHandler creates a handler that allows issuer to
// create new token types. If issuer is nil, anyone can create
// new tokens.
// TODO: check that permissioning???
func NewTokenHandler(auth x.Authenticator, issuer weave.Address) weave.Handler {
	return TokenHandler{
		auth:   auth,
		issuer: issuer,
		bucket: NewTokenBucket(),
	}
}

// NewSetNameHandler creates a handler that lets you set the
// name on a wallet one time.
func NewSetNameHandler(auth x.Authenticator, bucket NamedBucket) weave.Handler {
	return SetNameHandler{
		auth:   auth,
		bucket: bucket,
	}
}

// RegisterRoutes will instantiate and register
// all handlers in this package.
func RegisterRoutes(r weave.Registry, auth x.Authenticator, issuer weave.Address) {
	pathSend := cash.SendMsg{}.Path()
	r.Handle(pathSend, NewSendHandler(auth))
	r.Handle(pathNewTokenMsg, NewTokenHandler(auth, issuer))
	r.Handle(pathSetNameMsg, NewSetNameHandler(auth, NewWalletBucket()))
}

// RegisterQuery will register wallets as "/wallets"
// and tokens as "/tokens".
func RegisterQuery(qr weave.QueryRouter) {
	NewWalletBucket().Register("wallets", qr)
	NewTokenBucket().Register("tokens", qr)
}

// TokenHandler will handle creating new tokens.
type TokenHandler struct {
	auth   x.Authenticator
	bucket TickerBucket
	issuer weave.Address
}

var _ weave.Handler = TokenHandler{}

// Check just verifies it is properly formed and returns
// the cost of executing it.
func (h TokenHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	return &weave.CheckResult{GasAllocated: newTokenCost}, nil
}

// Deliver moves the tokens from sender to receiver if
// all preconditions are met.
func (h TokenHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	// make the token
	token := NewToken(msg.Ticker, msg.Name, msg.SigFigs)
	if err := h.bucket.Save(db, token); err != nil {
		return nil, err
	}
	return &weave.DeliverResult{}, err
}

// validate does all common pre-processing between Check and Deliver
func (h TokenHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*NewTokenMsg, error) {
	var msg NewTokenMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	// Make sure we have permission if the issuer is set.
	if h.issuer == nil || !h.auth.HasAddress(ctx, h.issuer) {
		return nil, errors.Wrapf(errors.ErrUnauthorized, "Token only issued by %s", h.issuer)
	}

	obj, err := h.bucket.Get(db, msg.Ticker)
	if err != nil {
		return nil, err
	}
	if obj != nil {
		return nil, errors.Wrapf(errors.ErrDuplicate, "token with ticker %s", msg.Ticker)
	}

	return &msg, nil
}

// SetNameHandler will set a name for objects in this bucket.
type SetNameHandler struct {
	auth   x.Authenticator
	bucket NamedBucket
}

var _ weave.Handler = SetNameHandler{}

// Check just verifies it is properly formed and returns
// the cost of executing it.
func (h SetNameHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	return &weave.CheckResult{GasAllocated: setNameCost}, nil
}

// Deliver moves the tokens from sender to receiver if
// all preconditions are met.
func (h SetNameHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	obj, err := h.bucket.Get(db, msg.Address)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, errors.Wrapf(errors.ErrNotFound, "wallet %s", msg.Address)
	}
	named := AsNamed(obj)
	err = named.SetName(msg.Name)
	if err != nil {
		return nil, err
	}

	if err := h.bucket.Save(db, obj); err != nil {
		return nil, err
	}
	return &weave.DeliverResult{}, nil
}

// validate does all common pre-processing between Check and Deliver.
func (h SetNameHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*SetWalletNameMsg, error) {
	var msg SetWalletNameMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	// Only wallet owner can set the name.
	if !h.auth.HasAddress(ctx, msg.Address) {
		return nil, errors.Wrap(errors.ErrUnauthorized, "Not wallet owner")
	}

	return &msg, nil
}
