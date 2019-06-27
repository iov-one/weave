package aswap

import (
	"bytes"
	"context"
	"crypto/sha256"

	"github.com/iov-one/weave"
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
	r = migration.SchemaMigratingRegistry("aswap", r)
	bucket := NewBucket()

	r.Handle(&CreateMsg{}, CreateSwapHandler{auth, bucket, cashctrl})
	r.Handle(&ReleaseMsg{}, ReleaseSwapHandler{auth, bucket, cashctrl})
	r.Handle(&ReturnSwapMsg{}, ReturnSwapHandler{auth, bucket, cashctrl})
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
func (h CreateSwapHandler) Check(ctx context.Context, info weave.BlockInfo, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
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
func (h CreateSwapHandler) Deliver(ctx context.Context, info weave.BlockInfo, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)

	if err != nil {
		return nil, err
	}

	// create a swap object
	swap := &Swap{
		Metadata:     msg.Metadata,
		Src:          msg.Src,
		Recipient:    msg.Recipient,
		Timeout:      msg.Timeout,
		Memo:         msg.Memo,
		PreimageHash: msg.PreimageHash,
	}

	obj, err := h.bucket.Create(db, swap)
	if err != nil {
		return nil, err
	}

	if err := cash.MoveCoins(db, h.bank, swap.Src, SwapAddr(obj.Key(), swap), msg.Amount); err != nil {
		return nil, err
	}

	// return id of swap to use in future calls
	res := &weave.DeliverResult{
		Data: obj.Key(),
	}
	return res, nil
}

// validate does all common pre-processing between Check and Deliver.
func (h CreateSwapHandler) validate(ctx context.Context, info weave.BlockInfo,
	db weave.KVStore, tx weave.Tx) (*CreateMsg, error) {
	var msg CreateMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	// Sender must authorize this
	if !h.auth.HasAddress(ctx, msg.Src) {
		return nil, errors.ErrUnauthorized
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
func (h ReleaseSwapHandler) Check(ctx context.Context, info weave.BlockInfo, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, _, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	return &weave.CheckResult{GasAllocated: releaseSwapCost}, nil
}

// Deliver moves the tokens from swap account to the receiver if
// all preconditions are met. When the swap account is empty it is deleted.
func (h ReleaseSwapHandler) Deliver(ctx context.Context, info weave.BlockInfo, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	swapID, swap, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	swapAddr := SwapAddr(swapID, swap)

	amount, err := h.bank.Balance(db, swapAddr)
	if err != nil {
		return nil, err
	}

	// withdraw the money from swap to recipient
	if err := cash.MoveCoins(db, h.bank, swapAddr, swap.Recipient, amount); err != nil {
		return nil, err
	}

	// Delete swap when empty.
	if err := h.bucket.Delete(db, swapID); err != nil {
		return nil, err
	}

	return &weave.DeliverResult{}, nil
}

// validate does all common pre-processing between Check and Deliver.
func (h ReleaseSwapHandler) validate(ctx context.Context, info weave.BlockInfo, db weave.KVStore, tx weave.Tx) ([]byte, *Swap, error) {
	var msg ReleaseMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}

	swap, err := loadSwap(h.bucket, db, msg.SwapID)
	if err != nil {
		return nil, nil, err
	}

	preimageHash := HashBytes(msg.Preimage)

	if !bytes.Equal(swap.PreimageHash, preimageHash) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "invalid preimageHash")
	}

	if weave.IsExpired(ctx, swap.Timeout) {
		return nil, nil, errors.Wrap(errors.ErrState, "swap is expired")
	}

	return msg.SwapID, swap, nil
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
func (h ReturnSwapHandler) Check(ctx context.Context, info weave.BlockInfo, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, _, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	return &weave.CheckResult{GasAllocated: returnSwapCost}, nil
}

// Deliver moves all the tokens from the swap to the defined sender if
// all preconditions are met. The swap is deleted afterwards.
func (h ReturnSwapHandler) Deliver(ctx context.Context, info weave.BlockInfo, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, swap, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	swapAddr := SwapAddr(msg.SwapID, swap)

	available, err := h.bank.Balance(db, swapAddr)
	if err != nil {
		return nil, err
	}

	// withdraw all coins from swap to the defined "sender"
	if err := cash.MoveCoins(db, h.bank, swapAddr, swap.Src, available); err != nil {
		return nil, err
	}
	if err := h.bucket.Delete(db, msg.SwapID); err != nil {
		return nil, err
	}

	return &weave.DeliverResult{}, nil
}

// validate does all common pre-processing between Check and Deliver.
func (h ReturnSwapHandler) validate(ctx context.Context, info weave.BlockInfo, db weave.KVStore, tx weave.Tx) (*ReturnSwapMsg, *Swap, error) {
	var msg ReturnSwapMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}

	swap, err := loadSwap(h.bucket, db, msg.SwapID)
	if err != nil {
		return nil, nil, err
	}

	if !weave.IsExpired(ctx, swap.Timeout) {
		return nil, nil, errors.Wrapf(errors.ErrState, "swap not expired %v", swap.Timeout)
	}

	return &msg, swap, nil
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

func HashBytes(preimage []byte) []byte {
	hash := sha256.Sum256(preimage)
	return hash[:]
}

func SwapAddr(key []byte, swap *Swap) weave.Address {
	swapAddrHash := bytes.Join([][]byte{key, swap.PreimageHash}, []byte("|"))
	// update swap address with a proper value
	return weave.NewCondition("aswap", "pre_hash", swapAddrHash).Address()
}
