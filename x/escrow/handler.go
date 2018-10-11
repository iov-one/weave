package escrow

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
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
	r.Handle(pathReturnEscrowMsg, ReturnEscrowHandler{auth, bucket, control})
	r.Handle(pathUpdateEscrowPartiesMsg, UpdateEscrowHandler{auth, bucket})
}

// RegisterQuery will register this bucket as "/escrows"
func RegisterQuery(qr weave.QueryRouter) {
	NewBucket().Register("escrows", qr)
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

	// apply a default for sender
	sender := weave.Address(msg.Src)
	if sender == nil {
		sender = x.MainSigner(ctx, h.auth).Address()
	}

	// create an escrow object
	escrow := &Escrow{
		Sender:    sender,
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
	dest := Condition(obj.Key()).Address()
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

	// verify that timeout is in the future
	height, _ := weave.GetHeight(ctx)
	if msg.Timeout <= height {
		return nil, ErrInvalidTimeout(msg.Timeout)
	}

	// sender must authorize this (if not set, defaults to MainSigner)
	if msg.Src != nil {
		sender := weave.Address(msg.Src)
		if !h.auth.HasAddress(ctx, sender) {
			return nil, errors.ErrUnauthorized()
		}
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
	msg, escrow, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	// use amount in message, or
	request := x.Coins(msg.Amount)
	available := x.Coins(escrow.Amount)
	if len(request) == 0 {
		request = available

		// TODO: add functionality to compare two sets
		// } else if !available.Contains(request) {
		// 	// ensure there is enough to pay
		// 	return res, cash.ErrInsufficientFunds()
	}

	// move the money from escrow to recipient
	key := msg.EscrowId
	sender := Condition(key).Address()
	dest := weave.Address(escrow.Recipient)
	for _, c := range request {
		err := h.cash.MoveCoins(db, sender, dest, *c)
		if err != nil {
			// this will rollback the half-finished tx
			return res, err
		}
		// remove coin from remaining balance
		available, err = available.Subtract(*c)
		if err != nil {
			return res, err
		}
	}

	// if there is something left, just update the balance...
	if available.IsPositive() {
		// return id as we can use again
		res.Data = key
		// this updates the object, as we have a pointer
		escrow.Amount = available
		obj := orm.NewSimpleObj(key, escrow)
		err = h.bucket.Save(db, obj)
	} else {
		// otherwise we finished the escrow and can delete it
		err = h.bucket.Delete(db, key)
	}

	// returns error if Save/Delete failed
	return res, err
}

// validate does all common pre-processing between Check and Deliver
func (h ReleaseEscrowHandler) validate(ctx weave.Context, db weave.KVStore,
	tx weave.Tx) (*ReleaseEscrowMsg, *Escrow, error) {

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

	escrow, err := loadEscrow(h.bucket, db, msg.EscrowId)
	if err != nil {
		return nil, nil, err
	}

	// arbiter or sender must authorize this
	arb := weave.Condition(escrow.Arbiter).Address()
	sender := weave.Address(escrow.Sender)
	if !h.auth.HasAddress(ctx, arb) && !h.auth.HasAddress(ctx, sender) {
		return nil, nil, errors.ErrUnauthorized()
	}

	// timeout must not have expired
	height, _ := weave.GetHeight(ctx)
	if escrow.Timeout < height {
		return nil, nil, ErrEscrowExpired(escrow.Timeout)
	}

	return msg, escrow, nil
}

//---- return

// ReturnEscrowHandler will set a name for objects in this bucket
type ReturnEscrowHandler struct {
	auth   x.Authenticator
	bucket Bucket
	cash   cash.Controller
}

var _ weave.Handler = ReturnEscrowHandler{}

// Check just verifies it is properly formed and returns
// the cost of executing it
func (h ReturnEscrowHandler) Check(ctx weave.Context, db weave.KVStore,
	tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	_, _, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	// return cost
	res.GasAllocated += returnEscrowCost
	return res, nil
}

// Deliver moves the tokens from sender to receiver if
// all preconditions are met
func (h ReturnEscrowHandler) Deliver(ctx weave.Context, db weave.KVStore,
	tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	key, escrow, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	// move the money from escrow to recipient
	sender := Condition(key).Address()
	dest := weave.Address(escrow.Sender)
	for _, c := range escrow.Amount {
		err := h.cash.MoveCoins(db, sender, dest, *c)
		if err != nil {
			// this will rollback the half-finished tx
			return res, err
		}
	}

	// now remove the finished escrow
	err = h.bucket.Delete(db, key)

	// returns error if Delete failed
	return res, err
}

// validate does all common pre-processing between Check and Deliver
func (h ReturnEscrowHandler) validate(ctx weave.Context, db weave.KVStore,
	tx weave.Tx) ([]byte, *Escrow, error) {

	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, nil, err
	}
	msg, ok := rmsg.(*ReturnEscrowMsg)
	if !ok {
		return nil, nil, errors.ErrUnknownTxType(rmsg)
	}

	err = msg.Validate()
	if err != nil {
		return nil, nil, err
	}

	// load escrow
	key := msg.EscrowId
	obj, err := h.bucket.Get(db, key)
	if err != nil {
		return nil, nil, err
	}
	escrow := AsEscrow(obj)
	if escrow == nil {
		return nil, nil, ErrNoSuchEscrow(key)
	}

	// timeout must have expired
	height, _ := weave.GetHeight(ctx)
	if height <= escrow.Timeout {
		return nil, nil, ErrEscrowNotExpired(escrow.Timeout)
	}

	return key, escrow, nil
}

//---- update

// UpdateEscrowHandler will set a name for objects in this bucket
type UpdateEscrowHandler struct {
	auth   x.Authenticator
	bucket Bucket
}

var _ weave.Handler = UpdateEscrowHandler{}

// Check just verifies it is properly formed and returns
// the cost of executing it
func (h UpdateEscrowHandler) Check(ctx weave.Context, db weave.KVStore,
	tx weave.Tx) (weave.CheckResult, error) {
	var res weave.CheckResult
	_, _, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	// return cost
	res.GasAllocated += updateEscrowCost
	return res, nil
}

// Deliver moves the tokens from sender to receiver if
// all preconditions are met
func (h UpdateEscrowHandler) Deliver(ctx weave.Context, db weave.KVStore,
	tx weave.Tx) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	msg, escrow, err := h.validate(ctx, db, tx)
	if err != nil {
		return res, err
	}

	// update the escrow with message values
	if msg.Sender != nil {
		escrow.Sender = msg.Sender
	}
	if msg.Recipient != nil {
		escrow.Recipient = msg.Recipient
	}
	if msg.Arbiter != nil {
		escrow.Arbiter = msg.Arbiter
	}

	// save the updated escrow
	key := msg.EscrowId
	obj := orm.NewSimpleObj(key, escrow)
	err = h.bucket.Save(db, obj)

	// returns error if Save failed
	return res, err
}

// validate does all common pre-processing between Check and Deliver
func (h UpdateEscrowHandler) validate(ctx weave.Context, db weave.KVStore,
	tx weave.Tx) (*UpdateEscrowPartiesMsg, *Escrow, error) {

	rmsg, err := tx.GetMsg()
	if err != nil {
		return nil, nil, err
	}
	msg, ok := rmsg.(*UpdateEscrowPartiesMsg)
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

	// timeout must not have expired
	height, _ := weave.GetHeight(ctx)
	if height > escrow.Timeout {
		return nil, nil, ErrEscrowExpired(escrow.Timeout)
	}

	// we must have the permission for the items we want to change
	if msg.Sender != nil {
		sender := weave.Address(escrow.Sender)
		if !h.auth.HasAddress(ctx, sender) {
			return nil, nil, errors.ErrUnauthorized()
		}
	}
	if msg.Recipient != nil {
		rcpt := weave.Address(escrow.Recipient)
		if !h.auth.HasAddress(ctx, rcpt) {
			return nil, nil, errors.ErrUnauthorized()
		}
	}
	if msg.Arbiter != nil {
		arbiter := weave.Condition(escrow.Arbiter).Address()
		if !h.auth.HasAddress(ctx, arbiter) {
			return nil, nil, errors.ErrUnauthorized()
		}
	}

	return msg, escrow, nil
}

// helpers

// load escrow and cast it, returns error if not present
func loadEscrow(bucket Bucket, db weave.KVStore, escrowID []byte) (*Escrow, error) {
	obj, err := bucket.Get(db, escrowID)
	if err != nil {
		return nil, err
	}
	escrow := AsEscrow(obj)
	if escrow == nil {
		return nil, ErrNoSuchEscrow(escrowID)
	}
	return escrow, nil
}
