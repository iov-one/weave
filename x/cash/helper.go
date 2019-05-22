package cash

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
)

func MoveCoins(db weave.KVStore, bank CoinMover, src, dest weave.Address, amounts []*coin.Coin) error {
	for _, c := range amounts {
		err := bank.MoveCoins(db, src, dest, *c)
		if err != nil {
			return errors.Wrapf(err, "failed to move %q", c.String())
		}
	}
	return nil
}
