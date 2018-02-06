package coins

import (
	"fmt"

	"github.com/confio/weave"
)

// MoveCoins moves the given amount from src to dest.
// If src doesn't exist, or doesn't have sufficient
// coins, it fails.
func MoveCoins(store weave.KVStore, src weave.Address,
	dest weave.Address, amount Coin) error {

	if !amount.IsPositive() {
		// TODO: better error
		return fmt.Errorf("MoveCoins must be positive")
	}

	sender := GetWallet(store, NewKey(src))
	if sender == nil {
		// TODO: better error
		return fmt.Errorf("Sender does not exist")
	}

	if !sender.Contains(amount) {
		// TODO: better error
		return fmt.Errorf("Sender does not have enough coins")
	}

	recipient := GetOrCreateWallet(store, NewKey(dest))
	sender.Subtract(amount)
	recipient.Add(amount)

	// make sure it didn't overflow
	if err := recipient.Validate(); err != nil {
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
	amount Coin) error {

	recipient := GetOrCreateWallet(store, NewKey(dest))
	recipient.Add(amount)

	// make sure it didn't overflow
	if err := recipient.Validate(); err != nil {
		return err
	}

	recipient.Save()
	return nil
}
