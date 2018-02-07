package coins

import (
	"fmt"

	"github.com/confio/weave/errors"
)

// ABCI Response Codes
// x/coins reserves 30 ~ 39.
const (
	CodeInvalidCurrency   uint32 = 30
	CodeInvalidCoin              = 31
	CodeInsufficientFees         = 32
	CodeInsufficientFunds        = 33
	CodeInvalidAmount            = 34
	CodeInvalidMemo              = 35
	CodeEmptyAccount             = 36
)

var (
	errInvalidCurrency   = fmt.Errorf("Invalid currency code")
	errOutOfRange        = fmt.Errorf("Overflow coin range")
	errMismatchedSign    = fmt.Errorf("Mismatched sign")
	errInvalidWallet     = fmt.Errorf("Invalid wallet")
	errInsufficientFees  = fmt.Errorf("Insufficient fees")
	errInsufficientFunds = fmt.Errorf("Insufficient funds")
	errInvalidAmount     = fmt.Errorf("Invalid amount")
	errInvalidMemo       = fmt.Errorf("Invalid memo")
	errEmptyAccount      = fmt.Errorf("Account empty")
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

func ErrInsufficientFees(coin Coin) error {
	msg := coin.String()
	return errors.WithLog(msg, errInsufficientFees, CodeInsufficientFees)
}
func IsInsufficientFeesErr(err error) bool {
	return errors.IsSameError(errInsufficientFees, err)
}

func ErrInsufficientFunds() error {
	return errors.WithCode(errInsufficientFunds, CodeInsufficientFunds)
}
func IsInsufficientFundsErr(err error) bool {
	return errors.IsSameError(errInsufficientFunds, err)
}

func ErrInvalidAmount(reason string) error {
	return errors.WithLog(reason, errInvalidAmount, CodeInvalidAmount)
}
func IsInvalidAmountErr(err error) bool {
	return errors.IsSameError(errInvalidAmount, err)
}

func ErrInvalidMemo(reason string) error {
	return errors.WithLog(reason, errInvalidMemo, CodeInvalidMemo)
}
func IsInvalidMemoErr(err error) bool {
	return errors.IsSameError(errInvalidMemo, err)
}

func ErrEmptyAccount(addr []byte) error {
	msg := fmt.Sprintf("%X", addr)
	return errors.WithLog(msg, errEmptyAccount, CodeEmptyAccount)
}
func IsEmptyAccountErr(err error) bool {
	return errors.IsSameError(errEmptyAccount, err)
}
