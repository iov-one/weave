package cash

import (
	"github.com/confio/weave"
	"github.com/confio/weave/x"
)

// Controller is the functionality needed by
// cash.Handler and cash.Decorator. BaseController
// should work plenty fine, but you can add other logic
// if so desired
type Controller interface {
	MoveCoins(store weave.KVStore, src weave.Address,
		dest weave.Address, amount x.Coin) error
	IssueCoins(store weave.KVStore, dest weave.Address,
		amount x.Coin) error
}

// BaseController is a simple implementation of controller
// wallet must return something that supports AsSet
type BaseController struct {
	bucket WalletBucket
}

// NewController returns a basic controller implementation
func NewController(bucket WalletBucket) BaseController {
	ValidateWalletBucket(bucket)
	return BaseController{bucket: bucket}
}

// MoveCoins moves the given amount from src to dest.
// If src doesn't exist, or doesn't have sufficient
// coins, it fails.
func (c BaseController) MoveCoins(store weave.KVStore,
	src weave.Address, dest weave.Address, amount x.Coin) error {

	if !amount.IsPositive() {
		return ErrInvalidAmount("Non-positive SendMsg")
	}

	sender, err := c.bucket.Get(store, src)
	if err != nil {
		return err
	}
	if sender == nil {
		return ErrEmptyAccount(src)
	}

	if !AsCoins(sender).Contains(amount) {
		return ErrInsufficientFunds()
	}

	recipient, err := c.bucket.GetOrCreate(store, dest)
	if err != nil {
		return err
	}
	err = Subtract(AsCoinage(sender), amount)
	if err != nil {
		return err
	}
	err = Add(AsCoinage(recipient), amount)
	if err != nil {
		return err
	}

	// save them and return
	err = c.bucket.Save(store, sender)
	if err != nil {
		return err
	}
	return c.bucket.Save(store, recipient)
}

// IssueCoins attempts to add the given amount of coins to
// the destination address. Fails if it overflows the wallet.
//
// Note the amount may also be negative:
// "the lord giveth and the lord taketh away"
func (c BaseController) IssueCoins(store weave.KVStore,
	dest weave.Address, amount x.Coin) error {

	recipient, err := c.bucket.GetOrCreate(store, dest)
	if err != nil {
		return err
	}
	err = Add(AsCoinage(recipient), amount)
	if err != nil {
		return err
	}

	return c.bucket.Save(store, recipient)
}
