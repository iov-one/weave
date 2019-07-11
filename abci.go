package weave

import (
	"fmt"

	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/common"
)

//---------- helpers for handling responses --------

// DeliverOrError returns an abci response for DeliverTx,
// converting the error message if present, or using the successful
// DeliverResult
func DeliverOrError(result *DeliverResult, err error, debug bool) abci.ResponseDeliverTx {
	if err != nil {
		return DeliverTxError(err, debug)
	}
	return result.ToABCI()
}

// CheckOrError returns an abci response for CheckTx,
// converting the error message if present, or using the successful
// CheckResult
func CheckOrError(result *CheckResult, err error, debug bool) abci.ResponseCheckTx {
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
	Diff []ValidatorUpdate
	// Tags, if present, will be used by tendermint to index and search the transaction history
	Tags []common.KVPair
	// GasUsed is currently unused field until effects in tendermint are clear
	GasUsed int64
}

// ToABCI converts our internal type into an abci response
func (d DeliverResult) ToABCI() abci.ResponseDeliverTx {
	return abci.ResponseDeliverTx{
		Data:    d.Data,
		Log:     d.Log,
		Tags:    d.Tags,
		GasUsed: d.GasUsed,
	}
}

// ParseDeliverOrError is the inverse of DeliverOrError
// It will parse back the abci response to return our internal format, or return an error on failed tx
func ParseDeliverOrError(res abci.ResponseDeliverTx) (*DeliverResult, error) {
	if res.Code != errors.SuccessABCICode {
		err := errors.ABCIError(res.Code, res.Log)
		return nil, err
	}
	return &DeliverResult{
		Data:    res.Data,
		Log:     res.Log,
		Tags:    res.Tags,
		GasUsed: res.GasUsed,
	}, nil
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

// DeliverTxError converts any error into a abci.ResponseDeliverTx, preserving
// as much info as possible.
// When in debug mode always the full error information is returned.
func DeliverTxError(err error, debug bool) abci.ResponseDeliverTx {
	code, log := errors.ABCIInfo(err, debug)
	if code != errors.SuccessABCICode {
		log = fmt.Sprintf("cannot deliver tx: %s", log)
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
	code, log := errors.ABCIInfo(err, debug)
	if code != errors.SuccessABCICode {
		log = fmt.Sprintf("cannot check tx: %s", log)
	}
	return abci.ResponseCheckTx{
		Code: code,
		Log:  log,
	}
}
