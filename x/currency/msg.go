package currency

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
)

var _ weave.Msg = (*NewTokenInfoMsg)(nil)

func (NewTokenInfoMsg) Path() string {
	return "currency/tokeninfo"
}

func (t *NewTokenInfoMsg) Validate() error {
	if !coin.IsCC(t.Ticker) {
		return coin.ErrInvalidCurrency.New(t.Ticker)
	}
	if !isTokenName(t.Name) {
		return errors.ErrInvalidState.Newf("invalid token name %v", t.Name)
	}
	return nil
}
