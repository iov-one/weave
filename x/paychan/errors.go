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

var (
	errInvalidAmount          = stderr.New("invalid amount")
	errInvalidMemo            = stderr.New("invalid memo")
	errInvalidSenderPublicKey = stderr.New("invalid sender public key")
	errInvalidSignature       = stderr.New("invalid signature")
	errInvalidTimeout         = stderr.New("invalid timeout")
	errInvalidTotal           = stderr.New("invalid total")
	errInvalidTransferred     = stderr.New("invalid transferred")
	errMissingRecipient       = stderr.New("missing recipient")
	errMissingSrc             = stderr.New("missing src")
	errMissingChannelID       = stderr.New("missing channel ID")
	errMissingSenderPubkey    = stderr.New("missing sender public key")
	errNoSuchPaymentChannel   = stderr.New("no such payment channel")
	errNotAllowed             = stderr.New("not allowed")
)

func ErrMissingChannelID() error {
	return errors.WithCode(errMissingChannelID, codeMissingCondition)
}

func IsMissingChannelIDErr(err error) bool {
	return errors.IsSameError(err, errMissingChannelID)
}

func ErrMissingRecipient() error {
	return errors.WithCode(errMissingRecipient, codeMissingCondition)
}

func IsMissingRecipientErr(err error) bool {
	return errors.IsSameError(err, errMissingRecipient)
}

func ErrMissingSrc() error {
	return errors.WithCode(errMissingSrc, codeMissingCondition)
}

func IsMissingSenderErr(err error) bool {
	return errors.IsSameError(err, errMissingSrc)
}

func ErrInvalidTimeout(timeout int64) error {
	msg := fmt.Sprint(timeout)
	return errors.WithLog(msg, errInvalidTimeout, codeInvalidCondition)
}

func ErrMissingSenderPubkey() error {
	return errors.WithCode(errMissingSenderPubkey, codeMissingCondition)
}

func IsMissingSenderPublicKeyErr(err error) bool {
	return errors.IsSameError(err, errMissingSenderPubkey)
}

func ErrInvalidSenderPublicKey() error {
	return errors.WithCode(errInvalidSenderPublicKey, codeInvalidCondition)
}

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

func IsInvalidTransferredErr(err error) bool {
	return errors.IsSameError(err, errInvalidTransferred)
}

func ErrInvalidMemo(memo string) error {
	return errors.WithLog(memo, errInvalidMemo, codeInvalidCondition)
}

func IsInvalidMemoErr(err error) bool {
	return errors.IsSameError(err, errInvalidMemo)
}

func ErrInvalidSignature() error {
	return errors.WithCode(errInvalidSignature, codeInvalidCondition)
}

func IsInvalidSignatureErr(err error) bool {
	return errors.IsSameError(err, errInvalidSignature)
}

func ErrNoSuchPaymentChannel(id []byte) error {
	s := hex.EncodeToString(id)
	return errors.WithLog(s, errNoSuchPaymentChannel, codeNotFound)
}

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

func IsInvalidAmountErr(err error) bool {
	return errors.IsSameError(err, errInvalidAmount)
}

func ErrNotAllowed(reason string) error {
	return errors.WithLog(reason, errNotAllowed, codeInvalidCondition)
}

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
