package escrow

import (
	"github.com/confio/weave"
	"github.com/confio/weave/errors"
	"github.com/confio/weave/x"
	"github.com/confio/weave/x/cash"
)

const (
	// pay escrow cost up-front
	createEscrowCost  int64 = 300
	returnEscrowCost  int64 = 0
	releaseEscrowCost int64 = 0
	updateEscrowCost  int64 = 50
)

// RegisterRoutes will instantiate and register
// all handlers in this package
func RegisterRoutes(r weave.Registry, auth x.Authenticator,
	control cash.Controller) {

	bucket := NewBucket()
	r.Handle(pathCreateEscrowMsg, CreateEscrowHandler{auth, bucket, control})
	// r.Handle(pathReleaseEscrowMsg, ReleaseEscrowHandler{auth, bucket, control})
	// r.Handle(pathReturnEscrowMsg, ReturnEscrowHandler{auth, bucket, control})
	// r.Handle(pathUpdateEscrowPartiesMsg, UpdateEscrowPartiesHandler{auth, bucket})
}

// CreateEscrowHandler will set a name for objects in this bucket
type CreateEscrowHandler struct {
	auth   x.Authenticator
	bucket Bucket
	cash   cash.Controller
}

var _ weave.Handler = CreateEscrowHandler{}

// Check just verifies it is properly formed and returns
// the cost of executing it
func (h CreateEscrowHandler) Check(ctx weave.Context, db weave.KVStore,
	tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	_, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	// return cost
	res.GasAllocated += createEscrowCost
	return res, nil
}

// Deliver moves the tokens from sender to receiver if
// all preconditions are met
func (h CreateEscrowHandler) Deliver(ctx weave.Context, db weave.KVStore,
	tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	// create an escrow object
	escrow := &Escrow{
		Sender:    msg.Sender,
		Arbiter:   msg.Arbiter,
		Recipient: msg.Recipient,
		Amount:    msg.Amount,
		Timeout:   msg.Timeout,
		Memo:      msg.Memo,
	}
	obj, err := h.bucket.Create(db, escrow)
	if err != nil {
		return res, err
	}

	// move the money to this object
	dest := Permission(obj.Key()).Address()
	sender := weave.Permission(msg.Sender).Address()
	for _, c := range escrow.Amount {
		err := h.cash.MoveCoins(db, sender, dest, *c)
		if err != nil {
			// this will rollback the half-finished tx
			return res, err
		}
	}

	// return id of escrow to use in future calls
	res.Data = obj.Key()
	return res, err
}

// validate does all common pre-processing between Check and Deliver
func (h CreateEscrowHandler) validate(ctx weave.Context, db weave.KVStore,
	tx weave.Tx) (*CreateEscrowMsg, error) {

	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}
	msg, ok := rmsg.(*CreateEscrowMsg)
	if !ok {
		return nil, errors.ErrUnknownTxType(rmsg)
	}

	err = msg.Validate()
	if err != nil {
		return nil, err
	}

	// sender must authorize this
	if !h.auth.HasAddress(ctx, msg.Sender) {
		return nil, errors.ErrUnauthorized()
	}

	// TODO: check balance? or just error on deliver?

	return msg, nil
}
