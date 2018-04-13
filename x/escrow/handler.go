package escrow

import (
	"github.com/confio/weave"
	"github.com/confio/weave/errors"
	"github.com/confio/weave/orm"
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
	r.Handle(pathReleaseEscrowMsg, ReleaseEscrowHandler{auth, bucket, control})
	// r.Handle(pathReturnEscrowMsg, ReturnEscrowHandler{auth, bucket, control})
	// r.Handle(pathUpdateEscrowPartiesMsg, UpdateEscrowPartiesHandler{auth, bucket})
}

//---- create

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
	sender := weave.Permission(msg.Sender).Address()
	if !h.auth.HasAddress(ctx, sender) {
		return nil, errors.ErrUnauthorized()
	}

	// TODO: check balance? or just error on deliver?

	return msg, nil
}

//---- release

// ReleaseEscrowHandler will set a name for objects in this bucket
type ReleaseEscrowHandler struct {
	auth   x.Authenticator
	bucket Bucket
	cash   cash.Controller
}

var _ weave.Handler = ReleaseEscrowHandler{}

// Check just verifies it is properly formed and returns
// the cost of executing it
func (h ReleaseEscrowHandler) Check(ctx weave.Context, db weave.KVStore,
	tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	_, _, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	// return cost
	res.GasAllocated += releaseEscrowCost
	return res, nil
}

// Deliver moves the tokens from sender to receiver if
// all preconditions are met
func (h ReleaseEscrowHandler) Deliver(ctx weave.Context, db weave.KVStore,
	tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, obj, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}
	escrow := AsEscrow(obj)

	// use amount in message, or
	request := x.Coins(msg.Amount)
	available := x.Coins(escrow.Amount)
	if len(request) == 0 {
		request = available.Clone()

		// TODO: add functionality to compare two sets
		// } else if !available.Contains(request) {
		// 	// ensure there is enough to pay
		// 	return res, cash.ErrInsufficientFunds()
	}

	// move the money from escrow to recipient
	sender := Permission(obj.Key()).Address()
	dest := weave.Permission(escrow.Recipient).Address()
	for _, c := range request {
		err := h.cash.MoveCoins(db, sender, dest, *c)
		if err != nil {
			// this will rollback the half-finished tx
			return res, err
		}
		available.Subtract(*c)
	}

	// if there is something left, just update the balance...
	if available.IsPositive() {
		// return id as we can use again
		res.Data = obj.Key()
		// this updates the object, as we have a pointer
		escrow.Amount = available
		err = h.bucket.Save(db, obj)
	} else {
		// otherwise we finished the escrow and can delete it
		err = h.bucket.Delete(db, obj.Key())
	}

	// returns error if Save/Delete failed
	return res, err
}

// validate does all common pre-processing between Check and Deliver
func (h ReleaseEscrowHandler) validate(ctx weave.Context, db weave.KVStore,
	tx weave.Tx) (*ReleaseEscrowMsg, orm.Object, error) {

	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, nil, err
	}
	msg, ok := rmsg.(*ReleaseEscrowMsg)
	if !ok {
		return nil, nil, errors.ErrUnknownTxType(rmsg)
	}

	err = msg.Validate()
	if err != nil {
		return nil, nil, err
	}

	// load escrow
	obj, err := h.bucket.Get(db, msg.EscrowId)
	if err != nil {
		return nil, nil, err
	}
	escrow := AsEscrow(obj)
	if escrow == nil {
		return nil, nil, ErrNoSuchEscrow(msg.EscrowId)
	}
	// arbiter must authorize this
	arbiter := weave.Permission(escrow.Arbiter).Address()
	if !h.auth.HasAddress(ctx, arbiter) {
		return nil, nil, errors.ErrUnauthorized()
	}

	return msg, obj, nil
}
