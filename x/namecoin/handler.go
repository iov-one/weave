package namecoin

import (
	"github.com/confio/weave"
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
	// TODO
	return nil
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
	r.Handle(pathNewTickerMsg, NewTokenHandler(auth, issuer))
	r.Handle(pathSetNameMsg, NewSetNameHandler(auth, bucket))
}
