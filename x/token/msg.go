package token

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
)

var _ weave.Msg = (*NewTokenInfoMsg)(nil)

func (NewTokenInfoMsg) Path() string {
	return "token/tokeninfo"
}

func (t *NewTokenInfoMsg) Validate() error {
	if !x.IsCC(t.Ticker) {
		return x.ErrInvalidCurrency(t.Ticker)
	}
	if !isTokenName(t.Name) {
		return ErrInvalidTokenName(t.Name)
	}
	if t.SigFigs < minSigFigs || t.SigFigs > maxSigFigs {
		return ErrInvalidSigFigs(t.SigFigs)
	}
	return nil
}
