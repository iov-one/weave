package currency

import (
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &CreateMsg{}, migration.NoModification)
}

func (CreateMsg) Path() string {
	return "currency/create"
}

func (msg *CreateMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	if !coin.IsCC(msg.Ticker) {
		errs = errors.AppendField(errs, "Ticker", errors.ErrCurrency)
	}
	if !isTokenName(msg.Name) {
		errs = errors.AppendField(errs, "Name", errors.ErrState)
	}
	return errs
}
