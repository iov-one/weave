package client

import (
	"context"
	"sync"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

// SubscribeTxByID will block until there is a result, then return it
// You must cancel the context to avoid blocking forever in some cases
func (c *Client) SubscribeTxByID(ctx context.Context, id TransactionID) (*CommitResult, error) {
	txs := make(chan CommitResult, 1)
	err := c.SubscribeTx(ctx, QueryTxByID(id), txs)
	if err != nil {
		return nil, err
	}

	// wait on first value... channel may be closed if subscription cancelled first
	res, ok := <-txs
	if !ok {
		return nil, errors.Wrap(errors.ErrTimeout, "unsubscribed before result")
	}
	return &res, nil
}

// WatchTx will block until this transaction makes it into a block
// It will return immediately if the id was included in a block prior to the query, to avoid timing issues
// You can use context.Context to pass in a timeout
func (c *Client) WatchTx(ctx context.Context, id TransactionID) (*CommitResult, error) {
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
	search, err := c.GetTxByID(ctx, id)
	if err != nil || search != nil {
		return search, err
	}

	// now we just wait until the subscription returns fruit
	result := <-sub
	return result.result, result.err
}

// CommitTx will block on both Check and Deliver, returning when it is in a block
func (c *Client) CommitTx(ctx context.Context, tx weave.Tx) (*CommitResult, error) {
	// This can be combined from other primatives
	check, err := c.SubmitTx(ctx, tx)
	if err != nil {
		return nil, err
	}
	if check.Err != nil {
		return check.AsCommitError(), nil
	}
	return c.WatchTx(ctx, check.ID)
}

// WatchTxs will watch a list of transactions in parallel
func (c *Client) WatchTxs(ctx context.Context, ids []TransactionID) ([]*CommitResult, error) {
	var mutex sync.Mutex
	// FIXME: make this multierror when that exists
	var gotErr error
	res := make([]*CommitResult, len(ids))

	wg := sync.WaitGroup{}
	for i, id := range ids {
		if id != nil {
			wg.Add(1)
			// pass as args to avoid using same variables in multiple routines
			go func(idx int, myid []byte) {
				r, err := c.WatchTx(ctx, myid)

				// storing these values outside of the go routine needs to be in a mutex
				mutex.Lock()
				res[idx] = r
				if err != nil {
					gotErr = err
				}
				mutex.Unlock()
				wg.Done()
			}(i, id)
		}
	}
	wg.Wait()

	if gotErr != nil {
		return nil, gotErr
	}
	return res, nil
}

// CommitTxs will submit many transactions and wait until they are all included in blocks.
// Ideally, all in the same block
func (c *Client) CommitTxs(ctx context.Context, txs []weave.Tx) ([]*CommitResult, error) {
	// first submit them all (some may error), this should be in order
	var err error
	checks := make([]*MempoolResult, len(txs))
	for i, tx := range txs {
		checks[i], err = c.SubmitTx(ctx, tx)
		if err != nil {
			return nil, err
		}
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
