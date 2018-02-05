package coins

import "github.com/confio/weave"

// MoveCoins moves the given amount from src to dest.
// If src doesn't exist, or doesn't have sufficient
// coins, it fails.
func MoveCoins(store weave.KVStore, src weave.Address,
	dest weave.Address, amount Coin) error {

	// TODO
	return nil
}

// IssueCoins attempts to add the given amount of coins to
// the destination address. Fails if it overflows the wallet.
//
// Note the amount may also be negative:
// "the lord giveth and the lord taketh away"
func IssueCoins(store weave.KVStore, dest weave.Address,
	amount Coin) error {

	// TODO
	return nil
}
