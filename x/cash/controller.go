package cash

import (
	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
)

// CoinsMover is an interface for moving coins between accounts.
type CoinMover interface {
	// Moving coins must happen from the source to the destination address.
	// Zero or negative values must result in an error.
	MoveCoins(store weave.KVStore, src weave.Address, dest weave.Address, amount coin.Coin) error
}

// CoinMinter is an interface to create new coins.
type CoinMinter interface {
	// CoinMint increase the number of funds on given account by a
	// specified amount.
	CoinMint(weave.KVStore, weave.Address, coin.Coin) error
}

// Balancer is an interface to query the amount of coins.
type Balancer interface {
	// Balance returns the amount of funds stored under given account address.
	Balance(weave.KVStore, weave.Address) (coin.Coins, error)
}

// Controller is the functionality needed by cash.Handler and cash.Decorator.
// BaseController should work plenty fine, but you can add other logic if so
// desired
type Controller interface {
	CoinMover
	Balancer
}

// BaseController implements Controller interface, using WalletBucket as the
// storage engine. Wallet must return something that supports AsSet.
type BaseController struct {
	bucket WalletBucket
}

var _ Controller = BaseController{}

// NewController returns a base controller implementation.
func NewController(bucket WalletBucket) BaseController {
	ValidateWalletBucket(bucket)
	return BaseController{bucket: bucket}
}

// Balance returns the amount of funds stored under given account address.
func (c BaseController) Balance(store weave.KVStore, src weave.Address) (coin.Coins, error) {
	state, err := c.bucket.Get(store, src)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get account state")
	}
	if state == nil {
		return nil, errors.Wrap(errors.ErrNotFound, "no account")
	}
	return AsCoins(state), nil
}

// MoveCoins moves the given amount from src to dest.
// If src doesn't exist, or doesn't have sufficient
// coins, it fails.
func (c BaseController) MoveCoins(store weave.KVStore,
	src weave.Address, dest weave.Address, amount coin.Coin) error {

	if amount.IsZero() {
		return errors.Wrap(errors.ErrAmount, "zero value")
	}
	if !amount.IsPositive() {
		return errors.Wrapf(errors.ErrAmount, "non-positive SendMsg: %#v", &amount)
	}

	// load sender, subtract funds, and save
	sender, err := c.bucket.Get(store, src)
	if err != nil {
		return err
	}
	if sender == nil {
		return errors.Wrapf(errors.ErrEmpty, "empty account %s", src)
	}
	if !AsCoins(sender).Contains(amount) {
		return errors.Wrap(errors.ErrAmount, "funds")
	}
	err = Subtract(AsCoinage(sender), amount)
	if err != nil {
		return err
	}
	err = c.bucket.Save(store, sender)
	if err != nil {
		return err
	}

	// load/create recipient, add funds, save
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

// CoinMint attempts to add the given amount of coins to
// the destination address. Fails if it overflows the wallet.
//
// Note the amount may also be negative:
// "the lord giveth and the lord taketh away"
func (c BaseController) CoinMint(store weave.KVStore,
	dest weave.Address, amount coin.Coin) error {

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
