package aswap

import (
	"crypto/sha256"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
	"github.com/tendermint/tendermint/libs/common"
)

const (
	// pay swap cost up-front
	createSwapCost  int64 = 300
	returnSwapCost  int64 = 0
	releaseSwapCost int64 = 0

	// currently set to two days
	day = time.Hour * 24
	// amount of days for minTimeout
	timeoutDays = 2
	minTimeout  = timeoutDays * day

	tagSwapId string = "swap-id"
	tagAction string = "action"
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

	// create a swap object
	swap := &Swap{
		Metadata:  &weave.Metadata{},
		Src:       msg.Src,
		Address:   weave.NewCondition("aswap", "pre_hash", msg.PreimageHash).Address(),
		Recipient: msg.Recipient,
		Timeout:   msg.Timeout,
		Memo:      msg.Memo,
	}
	obj, err := h.bucket.Build(db, msg.PreimageHash, swap)
	if err != nil {
		return nil, err
	}

	// deposit amounts
	senderAddr := swap.Src
	if err := moveCoins(db, h.bank, senderAddr, swap.Address, msg.Amount); err != nil {
		return nil, err
	}

	// return id of swap to use in future calls
	res := &weave.DeliverResult{
		Tags: []common.KVPair{
			{Key: []byte(tagSwapId), Value: obj.Key()},
			{Key: []byte(tagAction), Value: []byte("create-swap")},
		},
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
	if IsExpired(ctx, msg.Timeout.Add(-minTimeout)) {
		return nil, errors.Wrapf(errors.ErrInvalidInput,
			"timeout should be a minimum of %d days from now", timeoutDays)
	}
	if IsExpired(ctx, msg.Timeout) {
		return nil, errors.Wrap(errors.ErrInvalidInput, "timeout in the past")
	}

	// Sender must authorize this
	if !h.auth.HasAddress(ctx, msg.Src) {
		return nil, errors.ErrUnauthorized
	}

	// Leave the most expensive operation till we've sanity-checked everything else.
	_, err := loadSwap(h.bucket, db, msg.PreimageHash)
	switch {
	case err == nil:
		return nil, errors.Wrap(errors.ErrDuplicate, "swap with the same preimage")
	case !errors.ErrEmpty.Is(err):
		return nil, err
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
	preimageHash, swap, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	amount, err := h.bank.Balance(db, swap.Address)
	if err != nil {
		return nil, err
	}

	// withdraw the money from swap to recipient
	if err := moveCoins(db, h.bank, swap.Address, swap.Recipient, amount); err != nil {
		return nil, err
	}

	// Delete swap when empty.
	if err := h.bucket.Delete(db, preimageHash); err != nil {
		return nil, err
	}

	res := &weave.DeliverResult{
		Tags: []common.KVPair{
			{Key: []byte(tagSwapId), Value: preimageHash},
			{Key: []byte(tagAction), Value: []byte("release-swap")},
		},
	}
	return res, nil
}

// validate does all common pre-processing between Check and Deliver.
func (h ReleaseSwapHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) ([]byte, *Swap, error) {
	var msg ReleaseSwapMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}

	preimageHash := hashPreimage(msg.Preimage)
	swap, err := loadSwap(h.bucket, db, preimageHash)
	if err != nil {
		return nil, nil, err
	}

	return preimageHash, swap, nil
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
	msg, swap, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	available, err := h.bank.Balance(db, swap.Address)
	if err != nil {
		return nil, err
	}

	// withdraw all coins from swap to the defined "sender"
	if err := moveCoins(db, h.bank, swap.Address, swap.Src, available); err != nil {
		return nil, err
	}
	if err := h.bucket.Delete(db, msg.PreimageHash); err != nil {
		return nil, err
	}
	res := &weave.DeliverResult{
		Tags: []common.KVPair{
			{Key: []byte(tagSwapId), Value: msg.PreimageHash},
			{Key: []byte(tagAction), Value: []byte("return-swap")},
		},
	}
	return res, nil
}

// validate does all common pre-processing between Check and Deliver.
func (h ReturnSwapHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*ReturnSwapMsg, *Swap, error) {
	var msg ReturnSwapMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, nil, errors.Wrap(err, "load msg")
	}

	swap, err := loadSwap(h.bucket, db, msg.PreimageHash)
	if err != nil {
		return nil, nil, err
	}

	if !IsExpired(ctx, swap.Timeout) {
		return nil, nil, errors.Wrapf(errors.ErrInvalidState, "swap not expired %v", swap.Timeout)
	}

	return &msg, swap, nil
}

// loadSwap loads swap and casts it, returns error if not present.
func loadSwap(bucket Bucket, db weave.KVStore, preimageHash []byte) (*Swap, error) {
	obj, err := bucket.Get(db, preimageHash)
	if err != nil {
		return nil, err
	}
	swap := AsSwap(obj)
	if swap == nil {
		return nil, errors.Wrapf(errors.ErrEmpty, "swap %d", preimageHash)
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

func hashPreimage(preimage []byte) []byte {
	hash := sha256.Sum256(preimage)
	return hash[:]
}
