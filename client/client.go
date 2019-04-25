package client

import (
	"context"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	pubsub "github.com/tendermint/tendermint/libs/pubsub/query"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

var QueryNewBlockHeader = tmtypes.EventQueryNewBlockHeader

const txPerPage = 50

// Client is a tendermint client wrapped to provide
// simple access to the basic data structures used in weave
//
// Basic accessors are declared here.
// Higher-level API build around these basic accessors is defined on another level
type Client struct {
	conn rpcclient.Client
	// subscriber is a unique identifier for subscriptions
	subscriber string
}

// NewClient wraps a WeaveClient around an existing tendermint client connection.
func NewClient(conn rpcclient.Client) *Client {
	return &Client{
		conn: conn,
		// TODO: make this random
		subscriber: "weaveclient",
	}
}

func (c *Client) Status(ctx context.Context) (*Status, error) {
	// TODO: add context timeout here
	status, err := c.conn.Status()
	if err != nil {
		return nil, errors.Wrapf(errors.ErrNetwork, "status: %s", err.Error())
	}
	return &Status{
		Height:     status.SyncInfo.LatestBlockHeight,
		CatchingUp: status.SyncInfo.CatchingUp,
	}, nil
}

// SubmitTx will submit the tx to the mempool and then return with success or error
// You will need to use WatchTx (easily parallelizable) to get the result.
// CommitTx and CommitTxs provide helpers for common use cases
func (c *Client) SubmitTx(ctx context.Context, tx weave.Tx) (*MempoolResult, error) {
	bz, err := tx.Marshal()
	if err != nil {
		return nil, errors.Wrapf(errors.ErrInvalidMsg, "marshaling: %s", err.Error())
	}
	// TODO: timeout here
	res, err := c.conn.BroadcastTxSync(bz)
	if err != nil {
		return nil, errors.Wrapf(errors.ErrNetwork, "submit tx: %s", err.Error())
	}

	if res.Code != 0 {
		err = errors.ABCIError(res.Code, res.Log)
	}
	return &MempoolResult{
		ID:  res.Hash,
		Err: err,
	}, nil
}

// Query is meant to mirror the abci query interface exactly, so we can wrap it with app.ABCIStore
// This will give us state from the application
//
// TODO: provide other Query interface that accepts context for timeout??
func (c *Client) Query(query RequestQuery) ResponseQuery {
	res, err := c.conn.ABCIQueryWithOptions(query.Path, query.Data, rpcclient.ABCIQueryOptions{Height: query.Height, Prove: query.Prove})
	// network error reported as special error code
	if err != nil {
		code, log := errors.ABCIInfo(errors.Wrap(errors.ErrNetwork, err.Error()), false)
		return ResponseQuery{
			Code: code,
			Log:  log,
		}
	}
	return res.Response
}

// GetTxByID will return 0 or 1 results (nil or result value)
func (c *Client) GetTxByID(ctx context.Context, id TransactionID) (*CommitResult, error) {
	// TODO: add context timeout here
	tx, err := c.conn.Tx(id, false) // FIXME: use proofs sometime
	if err != nil {
		return nil, errors.Wrapf(errors.ErrNetwork, "get tx: %s", err.Error())
	}
	return resultTxToCommitResult(tx), nil
}

// SearchTx will search for all committed transactions that match a query,
// returning them as one large array.
// It returns an error if the subscription request failed.
func (c *Client) SearchTx(ctx context.Context, query TxQuery) ([]*CommitResult, error) {
	// TODO: return actual transaction content as well? not just ID and Result
	// TODO: add context timeout here
	// FIXME: use proofs sometime
	// FIXME: iterate over all search results and combine them if multiple pages
	search, err := c.conn.TxSearch(query, false, 1, txPerPage)
	if err != nil {
		return nil, errors.Wrapf(errors.ErrNetwork, "search tx: %s", err.Error())
	}

	results := make([]*CommitResult, len(search.Txs))
	for i, tx := range search.Txs {
		results[i] = resultTxToCommitResult(tx)
	}
	return results, nil
}

// SubscribeTx will subscribe to all transactions that match a query, writing them to the
// results channel as they arrive. It returns an error if the subscription request failed.
// Once subscriptions start, the continue until the context is closed (or network error)
func (c *Client) SubscribeTx(ctx context.Context, query TxQuery, results chan<- CommitResult) error {
	data, err := c.subscribe(ctx, query)
	if err != nil {
		return err
	}

	// start a go routine to parse the incoming data and feed to the results channel
	go func(in <-chan interface{}) {
		for elem := range in {
			// TODO: return actual transaction content as well? not just ID and Result
			// TODO: safer casting???
			val := elem.(tmtypes.EventDataTx)
			res := txResultToCommitResult(val.TxResult)
			results <- res
		}
		close(results)
	}(data)

	return nil
}

// subscribe should be used internally, it wraps conn.Subscribe and uses ctx.Done() to trigger Unsubscription
func (c *Client) subscribe(ctx context.Context, query string) (<-chan interface{}, error) {
	q, err := pubsub.New(query)
	if err != nil {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "Query '%s': %s", query, err.Error())
	}

	out := make(chan interface{}, 1)
	err = c.conn.Subscribe(ctx, c.subscriber, q, out)
	if err != nil {
		return nil, errors.Wrapf(errors.ErrNetwork, "Subscribe to '%s': %s", query, err.Error())
	}
	// listen for context canceled to unsubscribe
	// put all variables in local scope to prevent long-lived references
	go func(stop <-chan struct{}, sub string, q *pubsub.Query) {
		<-stop
		c.conn.Unsubscribe(context.Background(), sub, q)
	}(ctx.Done(), c.subscriber, q)

	return out, nil
}

func resultTxToCommitResult(tx *ctypes.ResultTx) *CommitResult {
	res, err := weave.ParseDeliverOrError(tx.TxResult)
	return &CommitResult{
		ID:     tx.Hash,
		Height: tx.Height,
		Result: res,
		Err:    err,
	}
}

func txResultToCommitResult(tx tmtypes.TxResult) CommitResult {
	res, err := weave.ParseDeliverOrError(tx.Result)
	return CommitResult{
		ID:     tx.Tx.Hash(),
		Height: tx.Height,
		Result: res,
		Err:    err,
	}
}
