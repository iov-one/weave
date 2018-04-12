package cash

import (
	"github.com/confio/weave"
	"github.com/confio/weave/errors"
	"github.com/confio/weave/x"
)

// RegisterRoutes will instantiate and register
// all handlers in this package
func RegisterRoutes(r weave.Registry, auth x.Authenticator,
	control Controller) {

	r.Handle(pathSendMsg, NewSendHandler(auth, control))
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
func (h SendHandler) Check(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.CheckResult, error) {

	// ensure type and validate...
	var res weave.CheckResult
	rmsg, err := tx.GetMsg()
	if err != nil {
		return res, err
	}
	msg, ok := rmsg.(*SendMsg)
	if !ok {
		return res, errors.ErrUnknownTxType(rmsg)
	}

	err = msg.Validate()
	if err != nil {
		return res, err
	}

	// make sure we have permission from the sender
	if !h.auth.HasAddress(ctx, msg.Src) {
		return res, errors.ErrUnauthorized()
	}

	// return cost
	res.GasAllocated += sendTxCost
	return res, nil
}

// Deliver moves the tokens from sender to receiver if
// all preconditions are met
func (h SendHandler) Deliver(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.DeliverResult, error) {

	// ensure type and validate...
	var res weave.DeliverResult
	rmsg, err := tx.GetMsg()
	if err != nil {
		return res, err
	}
	msg, ok := rmsg.(*SendMsg)
	if !ok {
		return res, errors.ErrUnknownTxType(rmsg)
	}

	err = msg.Validate()
	if err != nil {
		return res, err
	}

	// make sure we have permission from the sender
	if !h.auth.HasAddress(ctx, msg.Src) {
		return res, errors.ErrUnauthorized()
	}

	// move the money....
	err = h.control.MoveCoins(store, msg.Src, msg.Dest, *msg.Amount)
	if err != nil {
		return res, err
	}

	return res, nil
}
