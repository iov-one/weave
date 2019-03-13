package weave

import (
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/common"
)

//---------- helpers for handling responses --------

// DeliverOrError returns an abci response for DeliverTx,
// converting the error message if present, or using the successful
// DeliverResult
func DeliverOrError(result DeliverResult, err error, debug bool) abci.ResponseDeliverTx {
	if err != nil {
		return DeliverTxError(err, debug)
	}
	return result.ToABCI()
}

// CheckOrError returns an abci response for CheckTx,
// converting the error message if present, or using the successful
// CheckResult
func CheckOrError(result CheckResult, err error, debug bool) abci.ResponseCheckTx {
	if err != nil {
		return CheckTxError(err, debug)
	}
	return result.ToABCI()
}

//---------- results and some wrappers --------

// DeliverResult captures any non-error abci result
// to make sure people use error for error cases
type DeliverResult struct {
	// Data is a machine-parseable return value, like id of created entity
	Data []byte
	// Log is human-readable informational string
	Log string
	// RequiredFee can set an custom fee that must be paid for this transaction to be allowed to run.
	// This may enforced by a decorator, such as cash.DynamicFeeDecorator
	RequiredFee coin.Coin
	// Diff, if present, will apply to the Validator set in tendermint next block
	Diff []abci.ValidatorUpdate
	// Tags, if present, will be used by tendermint to index and search the transaction history
	Tags []common.KVPair
	// GasUsed is currently unused field until effects in tendermint are clear
	GasUsed int64
}

// ToABCI converts our internal type into an abci response
func (d DeliverResult) ToABCI() abci.ResponseDeliverTx {
	return abci.ResponseDeliverTx{
		Data: d.Data,
		Log:  d.Log,
		Tags: d.Tags,
	}
}

// CheckResult captures any non-error abci result
// to make sure people use error for error cases
type CheckResult struct {
	// Data is a machine-parseable return value, like id of created entity
	Data []byte
	// Log is human-readable informational string
	Log string
	// RequiredFee can set an custom fee that must be paid for this transaction to be allowed to run.
	// This may enforced by a decorator, such as cash.DynamicFeeDecorator
	RequiredFee coin.Coin
	// GasAllocated is the maximum units of work we allow this tx to perform
	GasAllocated int64
	// GasPayment is the total fees for this tx (or other source of payment)
	//TODO: Implement when tendermint implements this properly
	GasPayment int64
}

// NewCheck sets the gas used and the response data but no more info
// these are the most common info needed to be set by the Handler
func NewCheck(gasAllocated int64, log string) CheckResult {
	return CheckResult{
		GasAllocated: gasAllocated,
		Log:          log,
	}
}

// ToABCI converts our internal type into an abci response
func (c CheckResult) ToABCI() abci.ResponseCheckTx {
	return abci.ResponseCheckTx{
		Data:      c.Data,
		Log:       c.Log,
		GasWanted: c.GasAllocated,
	}
}

// TickResult allows the Ticker to modify the validator set
type TickResult struct {
	Diff []abci.ValidatorUpdate
}

// DeliverTxError converts any error into a abci.ResponseDeliverTx, preserving
// as much info as possible.
// When in debug mode always the full error information is returned.
func DeliverTxError(err error, debug bool) abci.ResponseDeliverTx {
	err = errors.Redact(err, debug)
	code, log := errors.ABCIInfo(err, debug)
	if code != 0 {
		log = "cannot deliver tx: " + log
	}
	return abci.ResponseDeliverTx{
		Code: code,
		Log:  log,
	}
}

// CheckTxError converts any error into a abci.ResponseCheckTx, preserving as
// much info as possible.
// When in debug mode always the full error information is returned.
func CheckTxError(err error, debug bool) abci.ResponseCheckTx {
	err = errors.Redact(err, debug)
	code, log := errors.ABCIInfo(err, debug)
	if code != 0 {
		log = "cannot check tx: " + log
	}
	return abci.ResponseCheckTx{
		Code: code,
		Log:  log,
	}
}
