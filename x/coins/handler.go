package coins

import (
	"fmt"

	"github.com/confio/weave"
)

// RegisterRoutes will register all handlers in
// this package
func RegisterRoutes(r weave.Registry) {
	r.Handle(pathSendMsg, NewSendHandler())
}

// SendHandler will handle sending coins
type SendHandler struct{}

var _ weave.Handler = SendHandler{}

// NewSendHandler creates a handler for SendMsg
func NewSendHandler() SendHandler {
	return SendHandler{}
}

// Check just verifies it is properly formed and returns
// the cost of executing it
func (h SendHandler) Check(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.CheckResult, error) {

	// ensure type and validate...
	var res weave.CheckResult
	msg, ok := tx.GetMsg().(*SendMsg)
	if !ok {
		return res, fmt.Errorf("Expected *SendMsg, got %v", tx.GetMsg())
	}
	err := msg.Validate()
	if err != nil {
		return res, err
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
	msg, ok := tx.GetMsg().(*SendMsg)
	if !ok {
		return res, fmt.Errorf("Expected *SendMsg, got %v", tx.GetMsg())
	}
	err := msg.Validate()
	if err != nil {
		return res, err
	}

	// move the money....
	err = MoveCoins(store, msg.Src, msg.Dest, *msg.Amount)
	if err != nil {
		return res, err
	}

	return res, nil
}
