package paychan

import (
	"encoding/hex"
	stderr "errors"
	"fmt"

	"github.com/iov-one/weave/errors"
	x "github.com/iov-one/weave/x"
)

// ABCI Response Codes
//
// paychan takes 1021-1029
const (
	codeMissingCondition = 1021
	codeInvalidCondition = 1022
	codeNotFound         = 1023
)

func ErrMissingRecipient() error {
	return errors.WithCode(errMissingRecipient, codeMissingCondition)
}

func IsMissingRecipientErr(err error) bool {
	return errors.IsSameError(err, errMissingRecipient)
}

var errMissingRecipient = stderr.New("missing recipient")

func ErrInvalidTimeout(timeout int64) error {
	msg := fmt.Sprint(timeout)
	return errors.WithLog(msg, errInvalidTimeout, codeInvalidCondition)
}

var errInvalidTimeout = stderr.New("invalid timeout")

func ErrMissingSenderPublicKey() error {
	return errors.WithCode(errMissingSenderPublicKey, codeMissingCondition)
}

var errMissingSenderPublicKey = stderr.New("missing sender public key")

func IsMissingSenderPublicKeyErr(err error) bool {
	return errors.IsSameError(err, errMissingSenderPublicKey)
}

func ErrInvalidSenderPublicKey() error {
	return errors.WithCode(errInvalidSenderPublicKey, codeInvalidCondition)
}

var errInvalidSenderPublicKey = stderr.New("invalid sender public key")

func IsInvalidSenderPublicKeyErr(err error) bool {
	return errors.IsSameError(err, errInvalidSenderPublicKey)
}

func ErrInvalidTotal(total *x.Coin) error {
	msg := "(nil)"
	if total != nil {
		msg = total.String()
	}
	return errors.WithLog(msg, errInvalidTotal, codeInvalidCondition)
}

var errInvalidTotal = stderr.New("invalid total")

func IsInvalidTotalErr(err error) bool {
	return errors.IsSameError(err, errInvalidTotal)
}

func ErrInvalidTransferred(trans *x.Coin) error {
	msg := "(nil)"
	if trans != nil {
		msg = trans.String()
	}
	return errors.WithLog(msg, errInvalidTransferred, codeInvalidCondition)
}

var errInvalidTransferred = stderr.New("invalid transferred")

func IsInvalidTransferredErr(err error) bool {
	return errors.IsSameError(err, errInvalidTransferred)
}

func ErrInvalidMemo(memo string) error {
	return errors.WithLog(memo, errInvalidMemo, codeInvalidCondition)
}

var errInvalidMemo = stderr.New("invalid memo")

func IsInvalidMemoErr(err error) bool {
	return errors.IsSameError(err, errInvalidMemo)
}

func ErrInvalidSignature() error {
	return errors.WithCode(errInvalidSignature, codeInvalidCondition)
}

var errInvalidSignature = stderr.New("invalid signature")

func IsInvalidSignatureErr(err error) bool {
	return errors.IsSameError(err, errInvalidSignature)
}

func ErrNoSuchPaymentChannel(id []byte) error {
	s := hex.EncodeToString(id)
	return errors.WithLog(s, errNoSuchPaymentChannel, codeNotFound)
}

var errNoSuchPaymentChannel = stderr.New("no such payment channel")

func IsNoSuchPaymentChannelErr(err error) bool {
	return errors.IsSameError(err, errNoSuchPaymentChannel)
}

func ErrInvalidAmount(c *x.Coin) error {
	msg := "(nil)"
	if c != nil {
		msg = c.String()
	}
	return errors.WithLog(msg, errInvalidAmount, codeInvalidCondition)
}

var errInvalidAmount = stderr.New("invalid amount")

func IsInvalidAmountErr(err error) bool {
	return errors.IsSameError(err, errInvalidAmount)
}

func ErrNotAllowed(reason string) error {
	return errors.WithLog(reason, errNotAllowed, codeInvalidCondition)
}

var errNotAllowed = stderr.New("not allowed")

func IsNotAllowedErr(err error) bool {
	return errors.IsSameError(err, errNotAllowed)
}

func IsMissingConditionErr(err error) bool {
	return errors.HasErrorCode(err, codeMissingCondition)
}

func IsInvalidConditionErr(err error) bool {
	return errors.HasErrorCode(err, codeInvalidCondition)
}

func IsNotFoundErr(err error) bool {
	return errors.HasErrorCode(err, codeNotFound)
}
