package client

import (
	"context"
	"sync"

	"github.com/iov-one/weave"
)

type Client struct {
}

type TransactionID []byte

type MempoolResult struct {
	ID     TransactionID
	Result *weave.CheckResult
	Err    error
}

func (a MempoolResult) AsCommitError() CommitResult {
	return CommitResult{
		ID:  a.ID,
		Err: a.Err,
	}
}

type CommitResult struct {
	ID     TransactionID
	Height int64
	Result *weave.DeliverResult
	Err    error
}

// SubmitTx will submit the tx to the mempool and then return with success or error
// You will need to use WatchTx (easily parallelizable) to get the result.
// CommitTx and CommitTxs provide helpers for common use cases
func (c *Client) SubmitTx(ctx context.Context, tx weave.Tx) MempoolResult {
	// TODO: submit to the node
	return MempoolResult{}
}

// SearchTxByID will return 0 or 1 results (nil or result value)
func (c *Client) SearchTxByID(ctx context.Context, id TransactionID) *CommitResult {
	// TODO: search
	return nil
}

// SubscribeTxByID will block until there is a result, then return it
// You must cancel the context to avoid blocking forever in some cases
func (c *Client) SubscribeTxByID(ctx context.Context, id TransactionID) CommitResult {
	// TODO: subscribe
	// TODO: how to handle context being cancelled???
	return CommitResult{}
}

// WatchTx will block until this transaction makes it into a block
// It will return immediately if the id was included in a block prior to the query, to avoid timing issues
// You can use context.Context to pass in a timeout
func (c *Client) WatchTx(ctx context.Context, id TransactionID) CommitResult {
	// TODO: combine subscribe tx and search tx (two other functions to be writen)
	subctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// start a subscription
	sub := make(chan CommitResult, 1)
	go func() {
		res := c.SubscribeTxByID(subctx, id)
		sub <- res
	}()

	// try to search and if successful, abort the subscription
	search := c.SearchTxByID(ctx, id)
	if search != nil {
		return *search
	}

	// now we just wait until the subscription returns fruit
	result := <-sub
	return result
}

// CommitTx will block on both Check and Deliver, returning when it is in a block
func (c *Client) CommitTx(ctx context.Context, tx weave.Tx) CommitResult {
	// This can be combined from other primatives
	check := c.SubmitTx(ctx, tx)
	if check.Err != nil {
		return check.AsCommitError()
	}
	return c.WatchTx(ctx, check.ID)
}

// WatchTxs will watch a list of transactions in parallel
func (c *Client) WatchTxs(ctx context.Context, ids []TransactionID) []CommitResult {
	res := make([]CommitResult, len(ids))
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

// CommitTxs will submit many transactions and wait until they are all included in blocks.
// Ideally, all in the same block
func (c *Client) CommitTxs(ctx context.Context, txs []weave.Tx) []CommitResult {
	// first submit them all (some may error), this should be in order
	checks := make([]MempoolResult, len(txs))
	for i, tx := range txs {
		checks[i] = c.SubmitTx(ctx, tx)
	}

	// make a list of all successful ones to block on Deliver (in parallel)
	ids := make([]TransactionID, len(txs))
	for i, check := range checks {
		if check.Err == nil {
			ids[i] = check.ID
		}
	}
	results := c.WatchTxs(ctx, ids)

	// now, we combine the check errors into the deliver results
	for i, check := range checks {
		if check.Err != nil {
			results[i] = check.AsCommitError()
		}
	}

	return results
}
