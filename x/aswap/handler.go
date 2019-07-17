package aswap

import (
	"bytes"
	"crypto/sha256"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
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
	r.Handle(&ReturnMsg{}, ReturnSwapHandler{auth, bucket, cashctrl})
}

// RegisterQuery will register this bucket as "/aswaps"
func RegisterQuery(qr weave.QueryRouter) {
	NewBucket().Register("aswaps", qr)
}

// CreateSwapHandler creates a swap
type CreateSwapHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
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

	key, err := swapSeq.NextVal(db)
	if err != nil {
		return nil, errors.Wrap(err, "cannot acquire key")
	}
	swap := &Swap{
		Metadata:     &weave.Metadata{Schema: 1},
		Source:       msg.Source,
		Destination:  msg.Destination,
		Timeout:      msg.Timeout,
		Memo:         msg.Memo,
		PreimageHash: msg.PreimageHash,
		Address:      swapAddr(key, msg.PreimageHash),
	}
	if _, err := h.bucket.Put(db, key, swap); err != nil {
		return nil, errors.Wrap(err, "cannot save swap entity")
	}
	if err := cash.MoveCoins(db, h.bank, swap.Source, swap.Address, msg.Amount); err != nil {
		return nil, errors.Wrap(err, "cannot deposit funds")
	}
	return &weave.DeliverResult{Data: key}, nil
}

func swapAddr(key []byte, preimageHash []byte) weave.Address {
	swapAddrHash := bytes.Join([][]byte{key, preimageHash}, []byte("|"))
	// update swap address with a proper value
	return weave.NewCondition("aswap", "pre_hash", swapAddrHash).Address()
}

// validate does all common pre-processing between Check and Deliver.
func (h CreateSwapHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*CreateMsg, error) {
	var msg CreateMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	// Sender must authorize this
	if !h.auth.HasAddress(ctx, msg.Source) {
		return nil, errors.ErrUnauthorized
	}

	return &msg, nil
}

// ReleaseSwapHandler releases the amount to destination.
type ReleaseSwapHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
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
	swapID, swap, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	amount, err := h.bank.Balance(db, swap.Address)
	if err != nil {
		return nil, err
	}

	// withdraw the money from swap to destination
	if err := cash.MoveCoins(db, h.bank, swap.Address, swap.Destination, amount); err != nil {
		return nil, err
	}

	// Delete swap when empty.
	if err := h.bucket.Delete(db, swapID); err != nil {
		return nil, err
	}

	return &weave.DeliverResult{}, nil
}

// validate does all common pre-processing between Check and Deliver.
func (h ReleaseSwapHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) ([]byte, *Swap, error) {
	var msg ReleaseMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}

	var swap Swap
	if err := h.bucket.One(db, msg.SwapID, &swap); err != nil {
		return nil, nil, errors.Wrap(err, "cannot load swap entity from the store")
	}

	preimageHash := HashBytes(msg.Preimage)

	if !bytes.Equal(swap.PreimageHash, preimageHash) {
		return nil, nil, errors.Wrap(errors.ErrUnauthorized, "invalid preimageHash")
	}

	if weave.IsExpired(ctx, swap.Timeout) {
		return nil, nil, errors.Wrap(errors.ErrState, "swap is expired")
	}

	return msg.SwapID, &swap, nil
}

// ReturnSwapHandler returns funds to the sender when swap timed out.
type ReturnSwapHandler struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
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
	msg, swap, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	available, err := h.bank.Balance(db, swap.Address)
	if err != nil {
		return nil, err
	}

	// withdraw all coins from swap to the defined "sender"
	if err := cash.MoveCoins(db, h.bank, swap.Address, swap.Source, available); err != nil {
		return nil, err
	}
	if err := h.bucket.Delete(db, msg.SwapID); err != nil {
		return nil, err
	}

	return &weave.DeliverResult{}, nil
}

// validate does all common pre-processing between Check and Deliver.
func (h ReturnSwapHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*ReturnMsg, *Swap, error) {
	var msg ReturnMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}

	var swap Swap
	if err := h.bucket.One(db, msg.SwapID, &swap); err != nil {
		return nil, nil, errors.Wrap(err, "cannot load swap entity from the store")
	}

	if !weave.IsExpired(ctx, swap.Timeout) {
		return nil, nil, errors.Wrapf(errors.ErrState, "swap not expired %v", swap.Timeout)
	}

	return &msg, &swap, nil
}

func HashBytes(preimage []byte) []byte {
	hash := sha256.Sum256(preimage)
	return hash[:]
}
