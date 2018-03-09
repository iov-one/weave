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

// RegisterRoutes will instantiate and register
// all handlers in this package
func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	pathSend := cash.SendMsg{}.Path()
	r.Handle(pathSend, NewSendHandler(auth))
	// TODO: new ticker handler
}
