package client

import (
	"context"

	"github.com/iov-one/weave"
)

type Client struct {
}

type TransactionId []byte

type AsyncResult struct {
	ID     TransactionId
	Result *weave.CheckResult
	Err    error
}

type SyncResult struct {
	ID     TransactionId
	Result *weave.DeliverResult
	Err    error
}

// ??? needed ???
type CommitResult struct {
	ID      TransactionId
	Check   *weave.CheckResult
	Deliver *weave.DeliverResult
	Err     error
}

// SubmitTx will block on both Check and Deliver, returning when it is in a block
func (c *Client) SubmitTx(ctx context.Context, tx weave.Tx) SyncResult {
	return SyncResult{}
}

// This just blocks on Check, then can
func (c *Client) SubmitTxAsync(ctx context.Context, tx weave.Tx) AsyncResult {
	return AsyncResult{}
}

// WatchTx will block until this transaction makes it into a block
// It will return immediatelt if the id was included in a block prior to the query, to avoid timing issues
// You can use context.Context to pass in a timeout
func (c *Client) WatchTx(ctx context.Context, id TransactionId) SyncResult {
	return SyncResult{}
}

// WatchTxs will watch a list of transactions in parallel
func (c *Client) WatchTxs(ctx context.Context, ids []TransactionId) []SyncResult {
	// TODO: use go routines
	res := make([]SyncResult, len(ids))
	for i, id := range ids {
		if id != nil {
			res[i] = c.WatchTx(ctx, id)
		}
	}
	return res
}

// SubmitTxs will submit many transactions and wait until they are all included in blocks.
// Ideally, all in the same block
func (c *Client) SubmitTxs(ctx context.Context, txs []weave.Tx) []SyncResult {
	// first submit them all (some may error), this should be in order
	checks := make([]AsyncResult, len(txs))
	for i, tx := range txs {
		checks[i] = c.SubmitTxAsync(ctx, tx)
	}

	// make a list of all successful ones to block on Deliver (in parallel)
	ids := make([]TransactionId, len(txs))
	for i, check := range checks {
		if check.Err == nil {
			ids[i] = check.ID
		}
	}
	results := c.WatchTxs(ctx, ids)

	// now, we combine the check errors into the deliver results
	for i, check := range checks {
		if check.Err != nil {
			results[i].ID = check.ID
			results[i].Err = check.Err
		}
	}

	return results
}
