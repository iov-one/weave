package currency

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &NewTokenInfoMsg{}, migration.NoModification)
}

var _ weave.Msg = (*NewTokenInfoMsg)(nil)

func (NewTokenInfoMsg) Path() string {
	return "currency/tokeninfo"
}

func (t *NewTokenInfoMsg) Validate() error {
	if err := t.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if !coin.IsCC(t.Ticker) {
		return errors.Wrapf(errors.ErrCurrency, "invalid ticker: %s", t.Ticker)
	}
	if !isTokenName(t.Name) {
		return errors.Wrapf(errors.ErrInvalidState, "invalid token name %v", t.Name)
	}
	return nil
}
