package x

import (
	"fmt"

	"github.com/iov-one/weave/errors"
)

// ABCI Response Codes
const (
	CodeInvalidCurrency uint32 = 30
	CodeInvalidCoin            = 31
)

var (
	errInvalidCurrency = fmt.Errorf("Invalid currency code")
	errOutOfRange      = fmt.Errorf("Overflow coin range")
	errMismatchedSign  = fmt.Errorf("Mismatched sign")
	errInvalidWallet   = fmt.Errorf("Invalid wallet")
)

// ErrInvalidCurrency takes one or two currencies
// that are not proper
func ErrInvalidCurrency(cur string, other ...string) error {
	// this is for mismatch
	if len(other) > 0 {
		cur += " vs. " + other[0]
	}
	return errors.WithLog(cur, errInvalidCurrency, CodeInvalidCurrency)
}
func IsInvalidCurrencyErr(err error) bool {
	return errors.IsSameError(errInvalidCurrency, err)
}

//------ various invalid signatures ----
// all will match IsInvalidSignatureError

func ErrOutOfRange(coin Coin) error {
	msg := coin.String()
	return errors.WithLog(msg, errOutOfRange, CodeInvalidCoin)
}
func ErrMismatchedSign(coin Coin) error {
	msg := coin.String()
	return errors.WithLog(msg, errMismatchedSign, CodeInvalidCoin)
}
func ErrInvalidWallet(msg string) error {
	return errors.WithLog(msg, errInvalidWallet, CodeInvalidCoin)
}
func IsInvalidCoinErr(err error) bool {
	return errors.HasErrorCode(err, CodeInvalidCoin)
}
