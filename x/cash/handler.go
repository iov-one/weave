package cash

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/x"
)

// RegisterRoutes will instantiate and register
// all handlers in this package
func RegisterRoutes(r weave.Registry, auth x.Authenticator, control Controller) {
	r = migration.SchemaMigratingRegistry("cash", r)

	r.Handle(&SendMsg{}, NewSendHandler(auth, control))
	r.Handle(&UpdateConfigurationMsg{}, NewConfigHandler(auth))
}

// RegisterQuery will register this bucket as "/wallets"
func RegisterQuery(qr weave.QueryRouter) {
	NewBucket().Register("wallets", qr)
}

// SendHandler will handle sending coins
type SendHandler struct {
	auth    x.Authenticator
	control Controller
}

var _ weave.Handler = SendHandler{}

// NewSendHandler creates a handler for SendMsg
func NewSendHandler(auth x.Authenticator, control Controller) SendHandler {
	return SendHandler{
		auth:    auth,
		control: control,
	}
}

// Check just verifies it is properly formed and returns
// the cost of executing it
func (h SendHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	var msg SendMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	// Make sure we have permission from the sources
	if !h.auth.HasAddress(ctx, msg.Source) {
		return nil, errors.Wrap(errors.ErrUnauthorized, "Account owner signature missing")
	}

	res := weave.CheckResult{
		GasAllocated: sendTxCost,
	}
	return &res, nil
}

// Deliver moves the tokens from source to receiver if
// all preconditions are met
func (h SendHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	var msg SendMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	// Make sure we have permission from the source.
	if !h.auth.HasAddress(ctx, msg.Source) {
		return nil, errors.Wrap(errors.ErrUnauthorized, "Account owner signature missing")
	}

	if err := h.control.MoveCoins(store, msg.Source, msg.Destination, *msg.Amount); err != nil {
		return nil, err
	}
	return &weave.DeliverResult{}, nil
}

func NewConfigHandler(auth x.Authenticator) weave.Handler {
	var conf Configuration
	return gconf.NewUpdateConfigurationHandler("cash", &conf, auth)
}
