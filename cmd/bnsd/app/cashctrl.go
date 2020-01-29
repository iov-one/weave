package bnsd

import (
	"bytes"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x/cash"
)

// BnsCashController wraps provided cash controller implementation with
// functionality specific to BNS.
func BnsCashController(c cash.Controller) cash.Controller {
	return &CashController{
		b:    cash.NewBucket(),
		ctrl: c,
	}
}

// CashController is a BNS specific cash.Controller implementation.
type CashController struct {
	b    cash.Bucket
	ctrl cash.Controller
}

func (c *CashController) MoveCoins(store weave.KVStore, src weave.Address, dest weave.Address, amount coin.Coin) error {
	if err := c.ctrl.MoveCoins(store, src, dest, amount); err != nil {
		return errors.Wrap(err, "move coins")
	}
	if dest.Equals(burnWallet) {
		empty := cash.NewWallet(burnWallet)
		if err := c.b.Save(store, empty); err != nil {
			return errors.Wrap(err, "cannot flush burn wallet")
		}
	}
	return nil
}

// Burn wallet as requested in https://github.com/iov-one/weave/issues/1140
//
// Any funds send to this wallet should be instantly removed from the system.
// This can be achieved by making sure that the wallet is always empty and even
// if any funds were sent to it, flush it instantly.
//
// This address is represented by iov1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqvnwh0u
var burnWallet = weave.Address(bytes.Repeat([]byte{0}, weave.AddressLength))

func (c *CashController) Balance(db weave.KVStore, a weave.Address) (coin.Coins, error) {
	return c.ctrl.Balance(db, a)
}
