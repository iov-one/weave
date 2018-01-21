package weave

import (
	abci "github.com/tendermint/abci/types"

	"github.com/confio/weave/errors"
)

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
