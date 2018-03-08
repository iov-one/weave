package cash

import (
	"github.com/confio/weave"
	"github.com/confio/weave/x"
)

// MoveCoins moves the given amount from src to dest.
// If src doesn't exist, or doesn't have sufficient
// coins, it fails.
func MoveCoins(store weave.KVStore, src weave.Address,
	dest weave.Address, amount x.Coin) error {

	// TODO: we don't create this every function call....
	bucket := NewBucket()

	if !amount.IsPositive() {
		return ErrInvalidAmount("Non-positive SendMsg")
	}

	sender, err := bucket.Get(store, src)
	if err != nil {
		return err
	}
	if sender == nil {
		return ErrEmptyAccount(src)
	}

	if !sender.Coins().Contains(amount) {
		return ErrInsufficientFunds()
	}

	recipient, err := bucket.GetOrCreate(store, dest)
	if err != nil {
		return err
	}
	err = sender.Subtract(amount)
	if err != nil {
		return err
	}
	err = recipient.Add(amount)
	if err != nil {
		return err
	}

	// save them and return
	err = bucket.Save(store, sender)
	if err != nil {
		return err
	}
	return bucket.Save(store, recipient)
}

// IssueCoins attempts to add the given amount of coins to
// the destination address. Fails if it overflows the wallet.
//
// Note the amount may also be negative:
// "the lord giveth and the lord taketh away"
func IssueCoins(store weave.KVStore, dest weave.Address,
	amount x.Coin) error {

	bucket := NewBucket()

	recipient, err := bucket.GetOrCreate(store, dest)
	if err != nil {
		return err
	}
	err = recipient.Add(amount)
	if err != nil {
		return err
	}

	return bucket.Save(store, recipient)
}
