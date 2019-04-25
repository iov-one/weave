package client

import (
	"fmt"

	"github.com/iov-one/weave"
	abci "github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	tmtypes "github.com/tendermint/tendermint/types"
)

// TransactionID is the hash used to identify the transaction
type TransactionID = cmn.HexBytes

// RequestQuery is used for the query interface to mirror the abci query interface
type RequestQuery = abci.RequestQuery

// ResponseQuery is used for the query interface to mirror the abci query interface
type ResponseQuery = abci.ResponseQuery

// TxQuery is some query to find transactions
type TxQuery = string

// MempoolResult is returned from the mempool (CheckTx)
// Result is only set on success codes, Err is set if it was a failure code
type MempoolResult struct {
	ID  TransactionID
	Err error
}

// AsCommitError will turn an errored MempoolResult into a CommitResult
func (a *MempoolResult) AsCommitError() *CommitResult {
	if a.Err == nil {
		panic("failed assertion: AsCommitError can onyl be called on errors")
	}
	return &CommitResult{
		ID:  a.ID,
		Err: a.Err,
	}
}

// CommitResult is returned from the block (DeliverTx)
// Result is only set on success codes, Err is set if it was a failure code
type CommitResult struct {
	ID     TransactionID
	Height int64
	Result *weave.DeliverResult
	Err    error
}

// Status is the current status of the node we connect to.
// Latest block height is a useful info
type Status struct {
	Height     int64
	CatchingUp bool
}

type Header = tmtypes.Header

// type Status = ctypes.ResultStatus
type GenesisDoc = tmtypes.GenesisDoc

type resultOrError struct {
	result *CommitResult
	err    error
}

// QueryTxByID makes a subscription string based on the transaction id
func QueryTxByID(id TransactionID) TxQuery {
	return fmt.Sprintf("%s='%s' AND %s='%X'", tmtypes.EventTypeKey, tmtypes.EventTx, tmtypes.TxHashKey, id)
}
