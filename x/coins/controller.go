package coins

import (
	"github.com/confio/weave"
	"github.com/confio/weave/x"
)

// MoveCoins moves the given amount from src to dest.
// If src doesn't exist, or doesn't have sufficient
// coins, it fails.
func MoveCoins(store weave.KVStore, src weave.Address,
	dest weave.Address, amount x.Coin) error {

	if !amount.IsPositive() {
		return ErrInvalidAmount("Non-positive SendMsg")
	}

	sender := GetWallet(store, NewKey(src))
	if sender == nil {
		return ErrEmptyAccount(src)
	}

	if !sender.Coins().Contains(amount) {
		return ErrInsufficientFunds()
	}

	recipient := GetOrCreateWallet(store, NewKey(dest))
	err := sender.Subtract(amount)
	if err != nil {
		return err
	}
	err = recipient.Add(amount)
	if err != nil {
		return err
	}

	// save them and return
	sender.Save()
	recipient.Save()
	return nil
}

// IssueCoins attempts to add the given amount of coins to
// the destination address. Fails if it overflows the wallet.
//
// Note the amount may also be negative:
// "the lord giveth and the lord taketh away"
func IssueCoins(store weave.KVStore, dest weave.Address,
	amount x.Coin) error {

	recipient := GetOrCreateWallet(store, NewKey(dest))
	err := recipient.Add(amount)
	if err != nil {
		return err
	}

	recipient.Save()
	return nil
}
