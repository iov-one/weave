package escrow

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
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
func RegisterRoutes(r weave.Registry, auth x.Authenticator, cashctrl cash.Controller) {
	r = migration.SchemaMigratingRegistry("escrow", r)
	bucket := NewBucket()

	r.Handle(pathCreateEscrowMsg, CreateEscrowHandler{auth, bucket, cashctrl})
	r.Handle(pathReleaseEscrowMsg, ReleaseEscrowHandler{auth, bucket, cashctrl})
	r.Handle(pathReturnEscrowMsg, ReturnEscrowHandler{auth, bucket, cashctrl})
	r.Handle(pathUpdateEscrowPartiesMsg, UpdateEscrowHandler{auth, bucket})
}

// RegisterQuery will register this bucket as "/escrows"
func RegisterQuery(qr weave.QueryRouter) {
	orm.Register(NewBucket(), "escrows", qr)
}

//---- create

// CreateEscrowHandler will set a name for objects in this bucket
type CreateEscrowHandler struct {
	auth   x.Authenticator
	bucket Bucket
	bank   cash.CoinMover
}

var _ weave.Handler = CreateEscrowHandler{}

// Check just verifies it is properly formed and returns
// the cost of executing it.
func (h CreateEscrowHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	res := &weave.CheckResult{
		GasAllocated: createEscrowCost,
	}
	return res, nil
}

// Deliver moves the tokens from sender to the escrow account if
// all preconditions are met.
func (h CreateEscrowHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	// apply a default for sender
	sender := msg.Src
	if sender == nil {
		sender = x.MainSigner(ctx, h.auth).Address()
	}

	// create an escrow object
	escrow := &Escrow{
		Metadata:  &weave.Metadata{},
		Sender:    sender,
		Arbiter:   msg.Arbiter,
		Recipient: msg.Recipient,
		Timeout:   msg.Timeout,
		Memo:      msg.Memo,
	}

	obj, err := h.bucket.Create(db, escrow)
	if err != nil {
		return nil, err
	}

	// deposit amounts
	escrowAddr := Condition(obj.Key()).Address()
	senderAddr := weave.Address(escrow.Sender)
	if err := cash.MoveCoins(db, h.bank, senderAddr, escrowAddr, msg.Amount); err != nil {
		return nil, err
	}
	// return id of escrow to use in future calls
	res := &weave.DeliverResult{
		Data: obj.Key(),
	}
	return res, nil
}

// validate does all common pre-processing between Check and Deliver.
func (h CreateEscrowHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*CreateEscrowMsg, error) {
	var msg CreateEscrowMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	if weave.IsExpired(ctx, msg.Timeout) {
		return nil, errors.Wrap(errors.ErrInput, "timeout in the past")
	}

	// Sender must authorize this (if not set, defaults to MainSigner).
	if msg.Src != nil {
		if !h.auth.HasAddress(ctx, msg.Src) {
			return nil, errors.ErrUnauthorized
		}
	}

	return &msg, nil
}

// ReleaseEscrowHandler will set a name for objects in this bucket.
type ReleaseEscrowHandler struct {
	auth   x.Authenticator
	bucket Bucket
	bank   cash.Controller
}

var _ weave.Handler = ReleaseEscrowHandler{}

// Check just verifies it is properly formed and returns
// the cost of executing it
func (h ReleaseEscrowHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, _, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	return &weave.CheckResult{GasAllocated: releaseEscrowCost}, nil
}

// Deliver moves the tokens from escrow account to the receiver if
// all preconditions are met. When the escrow account is empty it is deleted.
func (h ReleaseEscrowHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, escrow, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	// use amount in message, or
	request := coin.Coins(msg.Amount)
	key := msg.EscrowId
	escrowAddr := Condition(key).Address()
	if len(request) == 0 {
		available, err := h.bank.Balance(db, escrowAddr)
		if err != nil {
			return nil, err
		}
		request = available
	}

	// withdraw the money from escrow to recipient
	if err := cash.MoveCoins(db, h.bank, escrowAddr, escrow.Recipient, request); err != nil {
		return nil, err
	}

	remainingCoins, err := h.bank.Balance(db, escrowAddr)
	if err != nil {
		return nil, err
	}
	if remainingCoins.IsPositive() {
		return &weave.DeliverResult{Data: key}, nil
	}
	// Delete escrow when empty.
	if err := h.bucket.Delete(db, key); err != nil {
		return nil, err
	}
	return &weave.DeliverResult{}, nil
}

// validate does all common pre-processing between Check and Deliver.
func (h ReleaseEscrowHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*ReleaseEscrowMsg, *Escrow, error) {
	var msg ReleaseEscrowMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	escrow, err := loadEscrow(h.bucket, db, msg.EscrowId)
	if err != nil {
		return nil, nil, err
	}

	// Arbiter or sender must authorize this.
	if !h.auth.HasAddress(ctx, escrow.Arbiter) && !h.auth.HasAddress(ctx, escrow.Sender) {
		return nil, nil, errors.ErrUnauthorized
	}

	if weave.IsExpired(ctx, escrow.Timeout) {
		err := errors.Wrapf(errors.ErrExpired, "escrow expired %v", escrow.Timeout)
		return nil, nil, err
	}

	return &msg, escrow, nil
}

// ReturnEscrowHandler will set a name for objects in this bucket
type ReturnEscrowHandler struct {
	auth   x.Authenticator
	bucket Bucket
	bank   cash.Controller
}

var _ weave.Handler = ReturnEscrowHandler{}

// Check just verifies it is properly formed and returns
// the cost of executing it.
func (h ReturnEscrowHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, _, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	return &weave.CheckResult{GasAllocated: returnEscrowCost}, nil
}

// Deliver moves all the tokens from the escrow to the defined sender if
// all preconditions are met. The escrow is deleted afterwards.
func (h ReturnEscrowHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	key, escrow, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	escrowAddr := Condition(key).Address()
	available, err := h.bank.Balance(db, escrowAddr)
	if err != nil {
		return nil, err
	}

	// withdraw all coins from escrow to the defined "sender"
	dest := weave.Address(escrow.Sender)
	if err := cash.MoveCoins(db, h.bank, escrowAddr, dest, available); err != nil {
		return nil, err
	}
	if err := h.bucket.Delete(db, key); err != nil {
		return nil, err
	}
	return &weave.DeliverResult{}, nil
}

// validate does all common pre-processing between Check and Deliver.
func (h ReturnEscrowHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) ([]byte, *Escrow, error) {
	var msg ReturnEscrowMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}

	escrow, err := loadEscrow(h.bucket, db, msg.GetEscrowId())
	if err != nil {
		return nil, nil, err
	}

	if !weave.IsExpired(ctx, escrow.Timeout) {
		return nil, nil, errors.Wrapf(errors.ErrState, "escrow not expired %v", escrow.Timeout)
	}

	return msg.EscrowId, escrow, nil
}

// UpdateEscrowHandler will set a name for objects in this bucket.
type UpdateEscrowHandler struct {
	auth   x.Authenticator
	bucket Bucket
}

var _ weave.Handler = UpdateEscrowHandler{}

// Check just verifies it is properly formed and returns
// the cost of executing it.
func (h UpdateEscrowHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, _, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	return &weave.CheckResult{GasAllocated: updateEscrowCost}, nil
}

// Deliver updates the any of the sender, recipient or arbiter if
// all preconditions are met. No coins are moved.
func (h UpdateEscrowHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, escrow, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
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
	if _, err := h.bucket.Update(db, key, escrow); err != nil {
		return nil, err
	}
	return &weave.DeliverResult{}, nil
}

// validate does all common pre-processing between Check and Deliver.
func (h UpdateEscrowHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*UpdateEscrowPartiesMsg, *Escrow, error) {
	var msg UpdateEscrowPartiesMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}

	escrow, err := loadEscrow(h.bucket, db, msg.GetEscrowId())
	if err != nil {
		return nil, nil, err
	}

	if weave.IsExpired(ctx, escrow.Timeout) {
		return nil, nil, errors.Wrapf(errors.ErrExpired, "escrow expired %v", escrow.Timeout)
	}

	// we must have the permission for the items we want to change
	if msg.Sender != nil {
		sender := weave.Address(escrow.Sender)
		if !h.auth.HasAddress(ctx, sender) {
			return nil, nil, errors.ErrUnauthorized
		}
	}
	if msg.Recipient != nil {
		if !h.auth.HasAddress(ctx, escrow.Recipient) {
			return nil, nil, errors.ErrUnauthorized
		}
	}
	if msg.Arbiter != nil {
		if !h.auth.HasAddress(ctx, escrow.Arbiter) {
			return nil, nil, errors.ErrUnauthorized
		}
	}

	return &msg, escrow, nil
}

// loadEscrow loads escrow and cast it, returns error if not present.
func loadEscrow(bucket Bucket, db weave.KVStore, escrowID []byte) (*Escrow, error) {
	obj, err := bucket.Get(db, escrowID)
	if err != nil {
		return nil, err
	}
	escrow := AsEscrow(obj)
	if escrow == nil {
		return nil, errors.Wrapf(errors.ErrEmpty, "escrow %d", escrowID)
	}
	return escrow, nil
}
