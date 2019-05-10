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

// Header is a tendermint block header
type Header = tmtypes.Header

// GenesisDoc is the full tendermint genesis file
type GenesisDoc = tmtypes.GenesisDoc

type resultOrError struct {
	result *CommitResult
	err    error
}

// Option represents an option supplied to subscription
type Option interface {
	isOption()
}

// OptionCapacity is used for setting channel outCapacity for
// subscriptions
type OptionCapacity struct {
	Capacity int
}

func (_ OptionCapacity) isOption() {
	// just satisfies the interface
}

// QueryTxByID makes a subscription string based on the transaction id
func QueryTxByID(id TransactionID) TxQuery {
	return fmt.Sprintf("%s='%X'", tmtypes.TxHashKey, id)
}

// QueryForHeader is a subscription query for all new headers
func QueryForHeader() string {
	return queryForEvent(tmtypes.EventNewBlockHeader)
}

func queryForEvent(eventType string) string {
	return fmt.Sprintf("%s='%s'", tmtypes.EventTypeKey, eventType)
}
