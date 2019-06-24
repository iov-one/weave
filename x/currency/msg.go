package currency

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &CreateMsg{}, migration.NoModification)
}

var _ weave.Msg = (*CreateMsg)(nil)

func (CreateMsg) Path() string {
	return "currency/tokeninfo"
}

func (t *CreateMsg) Validate() error {
	if err := t.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if !coin.IsCC(t.Ticker) {
		return errors.Wrapf(errors.ErrCurrency, "invalid ticker: %s", t.Ticker)
	}
	if !isTokenName(t.Name) {
		return errors.Wrapf(errors.ErrState, "invalid token name %v", t.Name)
	}
	return nil
}
