package weave

import (
	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/go-wire/data"

	"github.com/confio/weave/errors"
)

//---------- helpers for handling responses --------

// DeliverOrError returns an abci response for DeliverTx,
// converting the error message if present, or using the successful
// DeliverResult
func DeliverOrError(result DeliverResult, err error) abci.ResponseDeliverTx {
	if err != nil {
		return DeliverTxError(err)
	}
	return result.ToABCI()
}

// CheckOrError returns an abci response for CheckTx,
// converting the error message if present, or using the successful
// CheckResult
func CheckOrError(result CheckResult, err error) abci.ResponseCheckTx {
	if err != nil {
		return CheckTxError(err)
	}
	return result.ToABCI()
}

//---------- results and some wrappers --------

// DeliverResult captures any non-error abci result
// to make sure people use error for error cases
type DeliverResult struct {
	Data    data.Bytes
	Log     string
	Diff    []*abci.Validator
	Tags    []*abci.KVPair
	GasUsed int64 // unused
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
	Data data.Bytes
	Log  string
	// GasAllocated is the maximum units of work we allow this tx to perform
	GasAllocated int64
	// GasPayment is the total fees for this tx (or other source of payment)
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
		Data: c.Data,
		Log:  c.Log,
		Gas:  c.GasAllocated,
		Fee:  c.GasPayment,
	}
}

//---------- type safe error converters --------

// DeliverTxError converts any error into a abci.ResponseDeliverTx,
// preserving as much info as possible if it was already
// a TMError
func DeliverTxError(err error) abci.ResponseDeliverTx {
	tm := errors.Wrap(err)
	return abci.ResponseDeliverTx{
		Code: tm.ABCICode(),
		Log:  tm.ABCILog(),
	}
}

// CheckTxError converts any error into a abci.ResponseCheckTx,
// preserving as much info as possible if it was already
// a TMError
func CheckTxError(err error) abci.ResponseCheckTx {
	tm := errors.Wrap(err)
	return abci.ResponseCheckTx{
		Code: tm.ABCICode(),
		Log:  tm.ABCILog(),
	}
}
