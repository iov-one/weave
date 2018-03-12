package namecoin

import (
	"github.com/confio/weave"
	"github.com/confio/weave/errors"
	"github.com/confio/weave/x"
	"github.com/confio/weave/x/cash"
)

// NewFeeDecorator customizes cash/FeeDecorator to use our
// WalletBucket
func NewFeeDecorator(auth x.Authenticator,
	min x.Coin) weave.Decorator {
	return cash.NewFeeDecorator(auth, NewController(), min)
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
	// TODO
	return nil
}

// RegisterRoutes will instantiate and register
// all handlers in this package
func RegisterRoutes(r weave.Registry, auth x.Authenticator, issuer weave.Address) {
	pathSend := cash.SendMsg{}.Path()
	bucket := NewWalletBucket()
	r.Handle(pathSend, NewSendHandler(auth))
	r.Handle(pathNewTokenMsg, NewTokenHandler(auth, issuer))
	r.Handle(pathSetNameMsg, NewSetNameHandler(auth, bucket))
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

	// set the token
	obj, err := h.bucket.GetOrCreate(db, msg.Ticker)
	if err != nil {
		return res, err
	}
	if obj != nil {
		return res, ErrDuplicateToken(msg.Ticker)
	}
	token := AsToken(obj)
	token.SigFigs = msg.SigFigs // TODO: defaults???
	token.Name = msg.Name

	err = h.bucket.Save(db, obj)
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
		return nil, errors.ErrUnknownTxType(rmsg)
	}

	err = msg.Validate()
	if err != nil {
		return nil, err
	}

	// make sure we have permission if the issuer is set
	if h.issuer != nil && !h.auth.HasPermission(ctx, h.issuer) {
		return nil, errors.ErrUnauthorized()
	}
	return msg, nil
}
