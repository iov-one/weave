package client

import (
	"context"
	"sync"

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

func (a AsyncResult) AsSyncError() SyncResult {
	return SyncResult{
		ID:  a.ID,
		Err: a.Err,
	}
}

type SyncResult struct {
	ID     TransactionId
	Height int64
	Result *weave.DeliverResult
	Err    error
}

// This just blocks on Check, then returns result or error.
// ID will always be set
func (c *Client) SubmitTxAsync(ctx context.Context, tx weave.Tx) AsyncResult {
	// TODO: submit to the node
	return AsyncResult{}
}

// SearchTxById will return 0 or 1 results (nil or result value)
func (c *Client) SearchTxById(ctx context.Context, id TransactionId) *SyncResult {
	// TODO: search
	return nil
}

// SubscribeTxById will block until there is a result, then return it
// You must cancel the context to avoid blocking forever in some cases
func (c *Client) SubscribeTxById(ctx context.Context, id TransactionId) SyncResult {
	// TODO: subscribe
	// TODO: how to handle context being cancelled???
	return SyncResult{}
}

// WatchTx will block until this transaction makes it into a block
// It will return immediatelt if the id was included in a block prior to the query, to avoid timing issues
// You can use context.Context to pass in a timeout
func (c *Client) WatchTx(ctx context.Context, id TransactionId) SyncResult {
	// TODO: combine subscribe tx and search tx (two other functions to be writen)
	subctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// start a subscription
	sub := make(chan SyncResult, 1)
	go func() {
		res := c.SubscribeTxById(subctx, id)
		sub <- res
	}()

	// try to search and if successful, abort the subscription
	search := c.SearchTxById(ctx, id)
	if search != nil {
		return *search
	}

	// now we just wait until the subscription returns fruit
	result := <-sub
	return result
}

// SubmitTx will block on both Check and Deliver, returning when it is in a block
func (c *Client) SubmitTx(ctx context.Context, tx weave.Tx) SyncResult {
	// This can be combined from other primatives
	check := c.SubmitTxAsync(ctx, tx)
	if check.Err != nil {
		return check.AsSyncError()
	}
	return c.WatchTx(ctx, check.ID)
}

// WatchTxs will watch a list of transactions in parallel
func (c *Client) WatchTxs(ctx context.Context, ids []TransactionId) []SyncResult {
	res := make([]SyncResult, len(ids))
	wg := sync.WaitGroup{}
	for i, id := range ids {
		if id != nil {
			wg.Add(1)
			// pass as args to avoid using same variables in multiple routines
			go func(idx int, myid []byte) {
				// all write to other location in slice, so should be safe
				res[idx] = c.WatchTx(ctx, myid)
				wg.Done()
			}(i, id)
		}
	}
	wg.Wait()
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
			results[i] = check.AsSyncError()
		}
	}

	return results
}
