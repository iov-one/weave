package namecoin

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
)

// NewFeeDecorator customizes cash/FeeDecorator to use our
// WalletBucket
func NewFeeDecorator(auth x.Authenticator) weave.Decorator {
	return cash.NewFeeDecorator(auth, NewController())
}

// NewSendHandler customizes cash/SendHandler to use our
// WalletBucket
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
// all handlers in this package
func RegisterRoutes(r weave.Registry, auth x.Authenticator, issuer weave.Address) {
	pathSend := cash.SendMsg{}.Path()
	r.Handle(pathSend, NewSendHandler(auth))
	r.Handle(pathNewTokenMsg, NewTokenHandler(auth, issuer))
	r.Handle(pathSetNameMsg, NewSetNameHandler(auth, NewWalletBucket()))
}

// RegisterQuery will register wallets as "/wallets"
// and tokens as "/tokens"
func RegisterQuery(qr weave.QueryRouter) {
	NewWalletBucket().Register("wallets", qr)
	NewTokenBucket().Register("tokens", qr)
}

// TokenHandler will handle creating new tokens
type TokenHandler struct {
	auth   x.Authenticator
	bucket TickerBucket
	issuer weave.Address
}

var _ weave.Handler = TokenHandler{}

// Check just verifies it is properly formed and returns
// the cost of executing it
func (h TokenHandler) Check(ctx weave.Context, db weave.KVStore,
	tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	_, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	// return cost
	res.GasAllocated += newTokenCost
	return res, nil
}

// Deliver moves the tokens from sender to receiver if
// all preconditions are met
func (h TokenHandler) Deliver(ctx weave.Context, db weave.KVStore,
	tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	// make the token
	token := NewToken(msg.Ticker, msg.Name, msg.SigFigs)
	err = h.bucket.Save(db, token)
	return res, err
}

// validate does all common pre-processing between Check and Deliver
func (h TokenHandler) validate(ctx weave.Context, db weave.KVStore,
	tx weave.Tx) (*NewTokenMsg, error) {

	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}
	msg, ok := rmsg.(*NewTokenMsg)
	if !ok {
		return nil, errors.WithType(errors.ErrInvalidMsg, rmsg)
	}

	err = msg.Validate()
	if err != nil {
		return nil, err
	}

	// make sure we have permission if the issuer is set
	if h.issuer != nil && !h.auth.HasAddress(ctx, h.issuer) {
		return nil, errors.ErrUnauthorized
	}

	// make sure no token there yet
	obj, err := h.bucket.Get(db, msg.Ticker)
	if err != nil {
		return nil, err
	}
	if obj != nil {
		return nil, errors.ErrDuplicate.Newf("token with ticker %s", msg.Ticker)
	}

	return msg, nil
}

// SetNameHandler will set a name for objects in this bucket
type SetNameHandler struct {
	auth   x.Authenticator
	bucket NamedBucket
}

var _ weave.Handler = SetNameHandler{}

// Check just verifies it is properly formed and returns
// the cost of executing it
func (h SetNameHandler) Check(ctx weave.Context, db weave.KVStore,
	tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	_, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	// return cost
	res.GasAllocated += setNameCost
	return res, nil
}

// Deliver moves the tokens from sender to receiver if
// all preconditions are met
func (h SetNameHandler) Deliver(ctx weave.Context, db weave.KVStore,
	tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	// set the token
	obj, err := h.bucket.Get(db, msg.Address)
	if err != nil {
		return res, err
	}
	if obj == nil {
		return res, errors.ErrNotFound.Newf("wallet %s", msg.Address)
	}
	named := AsNamed(obj)
	err = named.SetName(msg.Name)
	if err != nil {
		return res, err
	}

	err = h.bucket.Save(db, obj)
	return res, err
}

// validate does all common pre-processing between Check and Deliver
func (h SetNameHandler) validate(ctx weave.Context, db weave.KVStore,
	tx weave.Tx) (*SetWalletNameMsg, error) {

	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}
	msg, ok := rmsg.(*SetWalletNameMsg)
	if !ok {
		return nil, errors.WithType(errors.ErrInvalidMsg, rmsg)
	}

	err = msg.Validate()
	if err != nil {
		return nil, err
	}

	// only wallet owner can set the name
	if !h.auth.HasAddress(ctx, msg.Address) {
		return nil, errors.ErrUnauthorized
	}

	return msg, nil
}
