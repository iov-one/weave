package client

import (
	"context"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

type Header = tmtypes.Header
type Status = ctypes.ResultStatus
type GenesisDoc = tmtypes.GenesisDoc

var QueryNewBlockHeader = tmtypes.EventQueryNewBlockHeader

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
		subscriber: "tools-client",
	}
}

// SubmitTx will submit the tx to the mempool and then return with success or error
// You will need to use WatchTx (easily parallelizable) to get the result.
// CommitTx and CommitTxs provide helpers for common use cases
func (c *Client) SubmitTx(ctx context.Context, tx weave.Tx) (*MempoolResult, error) {
	bz, err := tx.Marshal()
	if err != nil {
		return nil, errors.Wrapf(errors.ErrInvalidMsg, "marshaling: %s", err.Error())
	}
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

// SearchTx will search for all committed transactions that match a query,
// returning them as one large array.
// It returns an error if the subscription request failed.
func (c *Client) SearchTx(ctx context.Context, query TxQuery) ([]*CommitResult, error) {
	// TODO: return actual transaction content as well? not just ID and Result
	return nil, nil
}

// SubscribeTx will subscribe to all transactions that match a query, writing them to the
// results channel as they arrive. It returns an error if the subscription request failed.
// Once subscriptions start, the continue until the context is closed (or network error)
func (c *Client) SubscribeTx(ctx context.Context, query TxQuery, results chan<- CommitResult) error {
	// TODO: return actual transaction content as well? not just ID and Result
	return nil
}
