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

	r.Handle(&CreateMsg{}, CreateEscrowHandler{auth, bucket, cashctrl})
	r.Handle(&ReleaseMsg{}, ReleaseEscrowHandler{auth, bucket, cashctrl})
	r.Handle(&ReturnMsg{}, ReturnEscrowHandler{auth, bucket, cashctrl})
	r.Handle(&UpdatePartiesMsg{}, UpdateEscrowHandler{auth, bucket})
}

// RegisterQuery will register this bucket as "/escrows"
func RegisterQuery(qr weave.QueryRouter) {
	NewBucket().Register("escrows", qr)
}

// CreateEscrowHandler will set a name for objects in this bucket
type CreateEscrowHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
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

// Deliver moves the tokens from source to the escrow account if all
// preconditions are met.
func (h CreateEscrowHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	// apply a default for source
	source := msg.Source
	if source == nil {
		source = x.MainSigner(ctx, h.auth).Address()
	}

	key, err := escrowSeq.NextVal(db)
	if err != nil {
		return nil, errors.Wrap(err, "cannot acquire key")
	}

	// create an escrow object
	escrow := &Escrow{
		Metadata:    &weave.Metadata{},
		Source:      source,
		Arbiter:     msg.Arbiter,
		Destination: msg.Destination,
		Timeout:     msg.Timeout,
		Memo:        msg.Memo,
		Address:     Condition(key).Address(),
	}
	if _, err := h.bucket.Put(db, key, escrow); err != nil {
		return nil, errors.Wrap(err, "cannot store escrow")
	}

	// Deposit to the escrow account.
	if err := cash.MoveCoins(db, h.bank, escrow.Source, escrow.Address, msg.Amount); err != nil {
		return nil, err
	}
	return &weave.DeliverResult{Data: key}, nil
}

// validate does all common pre-processing between Check and Deliver.
func (h CreateEscrowHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*CreateMsg, error) {
	var msg CreateMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	if weave.IsExpired(ctx, msg.Timeout) {
		return nil, errors.Wrap(errors.ErrInput, "timeout in the past")
	}

	// Source must authorize this (if not set, defaults to MainSigner).
	if msg.Source != nil {
		if !h.auth.HasAddress(ctx, msg.Source) {
			return nil, errors.ErrUnauthorized
		}
	}

	return &msg, nil
}

// ReleaseEscrowHandler will set a name for objects in this bucket.
type ReleaseEscrowHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
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
	if len(request) == 0 {
		available, err := h.bank.Balance(db, escrow.Address)
		if err != nil {
			return nil, err
		}
		request = available
	}

	// withdraw the money from escrow to recipient
	if err := cash.MoveCoins(db, h.bank, escrow.Address, escrow.Destination, request); err != nil {
		return nil, err
	}

	remainingCoins, err := h.bank.Balance(db, escrow.Address)
	if err != nil {
		return nil, err
	}
	if remainingCoins.IsPositive() {
		return &weave.DeliverResult{Data: msg.EscrowId}, nil
	}
	// Delete escrow when empty.
	if err := h.bucket.Delete(db, msg.EscrowId); err != nil {
		return nil, err
	}
	return &weave.DeliverResult{}, nil
}

// validate does all common pre-processing between Check and Deliver.
func (h ReleaseEscrowHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*ReleaseMsg, *Escrow, error) {
	var msg ReleaseMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	var escrow Escrow
	if err := h.bucket.One(db, msg.EscrowId, &escrow); err != nil {
		return nil, nil, errors.Wrap(err, "cannot load escrow from the store")
	}

	// Arbiter or source must authorize this.
	if !h.auth.HasAddress(ctx, escrow.Arbiter) && !h.auth.HasAddress(ctx, escrow.Source) {
		return nil, nil, errors.ErrUnauthorized
	}

	if weave.IsExpired(ctx, escrow.Timeout) {
		err := errors.Wrapf(errors.ErrExpired, "escrow expired %v", escrow.Timeout)
		return nil, nil, err
	}

	return &msg, &escrow, nil
}

// ReturnEscrowHandler will set a name for objects in this bucket
type ReturnEscrowHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
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

// Deliver moves all the tokens from the escrow to the defined source if
// all preconditions are met. The escrow is deleted afterwards.
func (h ReturnEscrowHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	key, escrow, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	available, err := h.bank.Balance(db, escrow.Address)
	if err != nil {
		return nil, err
	}

	// withdraw all coins from escrow to the defined "source"
	dest := weave.Address(escrow.Source)
	if err := cash.MoveCoins(db, h.bank, escrow.Address, dest, available); err != nil {
		return nil, err
	}
	if err := h.bucket.Delete(db, key); err != nil {
		return nil, err
	}
	return &weave.DeliverResult{}, nil
}

// validate does all common pre-processing between Check and Deliver.
func (h ReturnEscrowHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) ([]byte, *Escrow, error) {
	var msg ReturnMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}

	var escrow Escrow
	if err := h.bucket.One(db, msg.EscrowId, &escrow); err != nil {
		return nil, nil, errors.Wrap(err, "cannot load escrow from the store")
	}

	if !weave.IsExpired(ctx, escrow.Timeout) {
		return nil, nil, errors.Wrapf(errors.ErrState, "escrow not expired %v", escrow.Timeout)
	}

	return msg.EscrowId, &escrow, nil
}

// UpdateEscrowHandler will set a name for objects in this bucket.
type UpdateEscrowHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
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

// Deliver updates the any of the source, recipient or arbiter if
// all preconditions are met. No coins are moved.
func (h UpdateEscrowHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, escrow, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	// Update the escrow with message values.
	if msg.Source != nil {
		escrow.Source = msg.Source
	}
	if msg.Destination != nil {
		escrow.Destination = msg.Destination
	}
	if msg.Arbiter != nil {
		escrow.Arbiter = msg.Arbiter
	}

	// Save the updated escrow.
	if _, err := h.bucket.Put(db, msg.EscrowId, escrow); err != nil {
		return nil, errors.Wrap(err, "cannot save")
	}
	return &weave.DeliverResult{}, nil
}

// validate does all common pre-processing between Check and Deliver.
func (h UpdateEscrowHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*UpdatePartiesMsg, *Escrow, error) {
	var msg UpdatePartiesMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}

	var escrow Escrow
	if err := h.bucket.One(db, msg.EscrowId, &escrow); err != nil {
		return nil, nil, errors.Wrap(err, "cannot load escrow from the store")
	}

	if weave.IsExpired(ctx, escrow.Timeout) {
		return nil, nil, errors.Wrapf(errors.ErrExpired, "escrow expired %v", escrow.Timeout)
	}

	// we must have the permission for the items we want to change
	if msg.Source != nil {
		source := weave.Address(escrow.Source)
		if !h.auth.HasAddress(ctx, source) {
			return nil, nil, errors.ErrUnauthorized
		}
	}
	if msg.Destination != nil {
		if !h.auth.HasAddress(ctx, escrow.Destination) {
			return nil, nil, errors.ErrUnauthorized
		}
	}
	if msg.Arbiter != nil {
		if !h.auth.HasAddress(ctx, escrow.Arbiter) {
			return nil, nil, errors.ErrUnauthorized
		}
	}

	return &msg, &escrow, nil
}
