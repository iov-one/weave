package aswap

import (
	"bytes"
	"crypto/sha256"
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
)

const (
	// pay swap cost up-front
	createSwapCost  int64 = 300
	returnSwapCost  int64 = 0
	releaseSwapCost int64 = 0
)

// RegisterRoutes will instantiate and register
// all handlers in this package
func RegisterRoutes(r weave.Registry, auth x.Authenticator, cashctrl cash.Controller) {
	bucket := NewBucket()

	r.Handle(pathCreateSwap, migration.SchemaMigratingHandler("aswap", CreateSwapHandler{auth, bucket, cashctrl}))
	r.Handle(pathReleaseSwap, migration.SchemaMigratingHandler("aswap", ReleaseSwapHandler{auth, bucket, cashctrl}))
	r.Handle(pathReturnReturn, migration.SchemaMigratingHandler("aswap", ReturnSwapHandler{auth, bucket, cashctrl}))
}

// RegisterQuery will register this bucket as "/aswaps"
func RegisterQuery(qr weave.QueryRouter) {
	NewBucket().Register("aswaps", qr)
}

//---- create

// CreateSwapHandler creates a swap
type CreateSwapHandler struct {
	auth   x.Authenticator
	bucket Bucket
	bank   cash.CoinMover
}

var _ weave.Handler = CreateSwapHandler{}

// Check does the validation and sets the cost of the transaction
func (h CreateSwapHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	res := &weave.CheckResult{
		GasAllocated: createSwapCost,
	}
	return res, nil
}

// Deliver moves the tokens from sender to the swap account if all conditions are met.
func (h CreateSwapHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	// apply a default for sender
	sender := msg.Src
	if sender == nil {
		sender = x.MainSigner(ctx, h.auth).Address()
	}

	// create an swap object
	swap := &Swap{
		Metadata:     &weave.Metadata{},
		Src:          sender,
		PreimageHash: msg.PreimageHash,
		Recipient:    msg.Recipient,
		Timeout:      msg.Timeout,
		Memo:         msg.Memo,
	}
	obj, err := h.bucket.Build(db, swap)
	if err != nil {
		return nil, err
	}

	// deposit amounts
	swapAddr := Condition(obj.Key()).Address()
	senderAddr := swap.Src
	if err := moveCoins(db, h.bank, senderAddr, swapAddr, msg.Amount); err != nil {
		return nil, err
	}
	// return id of swap to use in future calls
	res := &weave.DeliverResult{
		Data: obj.Key(),
	}
	return res, h.bucket.Save(db, obj)
}

// validate does all common pre-processing between Check and Deliver.
func (h CreateSwapHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*CreateSwapMsg, error) {
	var msg CreateSwapMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	if IsExpired(ctx, msg.Timeout) {
		return nil, errors.Wrap(errors.ErrInvalidInput, "timeout in the past")
	}

	// Sender must authorize this (if not set, defaults to MainSigner).
	if msg.Src != nil {
		if !h.auth.HasAddress(ctx, msg.Src) {
			return nil, errors.ErrUnauthorized
		}
	}

	return &msg, nil
}

// ReleaseSwapHandler releases the amount to recipient.
type ReleaseSwapHandler struct {
	auth   x.Authenticator
	bucket Bucket
	bank   cash.Controller
}

var _ weave.Handler = ReleaseSwapHandler{}

// Check just verifies it is properly formed and returns
// the cost of executing it
func (h ReleaseSwapHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, _, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	return &weave.CheckResult{GasAllocated: releaseSwapCost}, nil
}

// Deliver moves the tokens from swap account to the receiver if
// all preconditions are met. When the swap account is empty it is deleted.
func (h ReleaseSwapHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, swap, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	key := msg.SwapID
	swapAddr := Condition(key).Address()

	amount, err := h.bank.Balance(db, swapAddr)
	if err != nil {
		return nil, err
	}

	// withdraw the money from swap to recipient
	if err := moveCoins(db, h.bank, swapAddr, swap.Recipient, amount); err != nil {
		return nil, err
	}

	// Delete swap when empty.
	if err := h.bucket.Delete(db, key); err != nil {
		return nil, err
	}
	return &weave.DeliverResult{}, nil
}

// validate does all common pre-processing between Check and Deliver.
func (h ReleaseSwapHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*ReleaseSwapMsg, *Swap, error) {
	var msg ReleaseSwapMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}
	swap, err := loadSwap(h.bucket, db, msg.SwapID)
	if err != nil {
		return nil, nil, err
	}

	// Sender must authorize this.
	if !h.auth.HasAddress(ctx, swap.Src) {
		return nil, nil, errors.ErrUnauthorized
	}

	if !bytes.Equal(swap.PreimageHash, sha256.Sum256(msg.Preimage)[:]) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "invalid preimage")
	}

	if IsExpired(ctx, swap.Timeout) {
		err := errors.Wrapf(errors.ErrExpired, "swap expired %v", swap.Timeout)
		return nil, nil, err
	}

	return &msg, swap, nil
}

// ReturnSwapHandler returns funds to the sender when swap timed out.
type ReturnSwapHandler struct {
	auth   x.Authenticator
	bucket Bucket
	bank   cash.Controller
}

var _ weave.Handler = ReturnSwapHandler{}

// Check just verifies it is properly formed and returns
// the cost of executing it.
func (h ReturnSwapHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, _, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	return &weave.CheckResult{GasAllocated: returnSwapCost}, nil
}

// Deliver moves all the tokens from the swap to the defined sender if
// all preconditions are met. The swap is deleted afterwards.
func (h ReturnSwapHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	key, swap, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	swapAddr := Condition(key).Address()
	available, err := h.bank.Balance(db, swapAddr)
	if err != nil {
		return nil, err
	}

	// withdraw all coins from swap to the defined "sender"
	if err := moveCoins(db, h.bank, swapAddr, swap.Src, available); err != nil {
		return nil, err
	}
	if err := h.bucket.Delete(db, key); err != nil {
		return nil, err
	}
	return &weave.DeliverResult{}, nil
}


// validate does all common pre-processing between Check and Deliver.
// TODO: Do we need to check who initiates this? I would assume this would be the sender
// on the other hand I see no reasonable scenarios for abuse here, given the fee and the inability
// to supply any parameters except for valid swapID
func (h ReturnSwapHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) ([]byte, *Swap, error) {
	var msg ReturnSwapMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}

	swap, err := loadSwap(h.bucket, db, msg.SwapID)
	if err != nil {
		return nil, nil, err
	}

	if !IsExpired(ctx, swap.Timeout) {
		return nil, nil, errors.Wrapf(errors.ErrInvalidState, "swap not expired %v", swap.Timeout)
	}

	return msg.SwapID, swap, nil
}

// loadSwap loads swap and casts it, returns error if not present.
func loadSwap(bucket Bucket, db weave.KVStore, swapID []byte) (*Swap, error) {
	obj, err := bucket.Get(db, swapID)
	if err != nil {
		return nil, err
	}
	swap := AsSwap(obj)
	if swap == nil {
		return nil, errors.Wrapf(errors.ErrEmpty, "swap %d", swapID)
	}
	return swap, nil
}

func moveCoins(db weave.KVStore, bank cash.CoinMover, src, dest weave.Address, amounts []*coin.Coin) error {
	for _, c := range amounts {
		err := bank.MoveCoins(db, src, dest, *c)
		if err != nil {
			return errors.Wrapf(err, "failed to move %q", c.String())
		}
	}
	return nil
}
