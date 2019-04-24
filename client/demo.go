package client

import (
	"context"
	"sync"

	"github.com/iov-one/weave"
)

type resultOrError struct {
	result CommitResult
	err    error
}

// WatchTx will block until this transaction makes it into a block
// It will return immediately if the id was included in a block prior to the query, to avoid timing issues
// You can use context.Context to pass in a timeout
func (c *Client) WatchTx(ctx context.Context, id TransactionID) (CommitResult, error) {
	// TODO: combine subscribe tx and search tx (two other functions to be writen)
	subctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// start a subscription
	sub := make(chan resultOrError, 1)
	go func() {
		res, err := c.SubscribeTxByID(subctx, id)
		sub <- resultOrError{
			result: res,
			err:    err,
		}
	}()

	// try to search and if successful, abort the subscription
	search, err := c.SearchTxByID(ctx, id)
	if err != nil {
		return CommitResult{}, err
	}
	if search != nil {
		return *search, nil
	}

	// now we just wait until the subscription returns fruit
	result := <-sub
	return result.result, result.err
}

// CommitTx will block on both Check and Deliver, returning when it is in a block
func (c *Client) CommitTx(ctx context.Context, tx weave.Tx) (CommitResult, error) {
	// This can be combined from other primatives
	check := c.SubmitTx(ctx, tx)
	if check.Err != nil {
		return check.AsCommitError(), nil
	}
	return c.WatchTx(ctx, check.ID)
}

// WatchTxs will watch a list of transactions in parallel
func (c *Client) WatchTxs(ctx context.Context, ids []TransactionID) ([]CommitResult, error) {
	var err error
	res := make([]CommitResult, len(ids))

	wg := sync.WaitGroup{}
	for i, id := range ids {
		if id != nil {
			wg.Add(1)
			// pass as args to avoid using same variables in multiple routines
			go func(idx int, myid []byte) {
				// all write to other location in slice, so should be safe
				res[idx], err = c.WatchTx(ctx, myid)
				wg.Done()
			}(i, id)
		}
	}
	wg.Wait()

	if err != nil {
		return nil, err
	}
	return res, nil
}

// CommitTxs will submit many transactions and wait until they are all included in blocks.
// Ideally, all in the same block
func (c *Client) CommitTxs(ctx context.Context, txs []weave.Tx) ([]CommitResult, error) {
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
	results, err := c.WatchTxs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// now, we combine the check errors into the deliver results
	for i, check := range checks {
		if check.Err != nil {
			results[i] = check.AsCommitError()
		}
	}

	return results, nil
}
