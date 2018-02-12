package coins

import (
	"github.com/confio/weave"
	"github.com/confio/weave/errors"
)

// RegisterRoutes will instantiate and register
// all handlers in this package
func RegisterRoutes(r weave.Registry, auth weave.AuthFunc) {
	r.Handle(pathSendMsg, NewSendHandler(auth))
}

// SendHandler will handle sending coins
type SendHandler struct {
	auth weave.AuthFunc
}

var _ weave.Handler = SendHandler{}

// NewSendHandler creates a handler for SendMsg
func NewSendHandler(auth weave.AuthFunc) SendHandler {
	return SendHandler{
		auth: auth,
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
	if !weave.HasSigner(msg.Src, h.auth(ctx)) {
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
	// fmt.Printf("signers: %v\n", h.auth(ctx))
	// fmt.Printf("src: %v\n", msg.Src)
	if !weave.HasSigner(msg.Src, h.auth(ctx)) {
		return res, errors.ErrUnauthorized()
	}

	// move the money....
	err = MoveCoins(store, msg.Src, msg.Dest, *msg.Amount)
	if err != nil {
		return res, err
	}

	return res, nil
}
