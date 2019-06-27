package client

import (
	"context"
	"sync"
	"time"

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
	// we get an error for not found.... TODO: handle that differently
	search, _ := c.GetTxByID(ctx, id)
	if search != nil {
		return search, nil
	}

	// now we just wait until the subscription returns fruit
	result := <-sub
	return result.result, result.err
}

// CommitTx will block on both Check and Deliver, returning when it is in a block
func (c *Client) CommitTx(ctx context.Context, tx weave.Tx) (*CommitResult, error) {
	// This can be combined from other primitives
	check, err := c.SubmitTx(ctx, tx)
	if err != nil {
		return nil, err
	}
	res, err := c.WatchTx(ctx, check)
	if err == nil {
		// on success wait a bit so index is updated
		c.waitForTxIndex()
	}
	return res, err
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
// Ideally, all in the same block.
//
// If any tx fails in mempool or network, this returns an error
// TODO: improve error handling (maybe we need to use CommitResultOrError type?)
func (c *Client) CommitTxs(ctx context.Context, txs []weave.Tx) ([]*CommitResult, error) {
	// first submit them all (some may error), this should be in order
	var err error
	ids := make([]TransactionID, len(txs))
	for i, tx := range txs {
		ids[i], err = c.SubmitTx(ctx, tx)
		if err != nil {
			return nil, err
		}
	}

	results, err := c.WatchTxs(ctx, ids)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// WaitForNextBlock will return the next block header to arrive (as subscription)
func (c *Client) WaitForNextBlock(ctx context.Context) (*Header, error) {
	// ensure we close subscription at function return
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()

	headers := make(chan Header, 1)
	err := c.SubscribeHeaders(cctx, headers)
	if err != nil {
		return nil, err
	}

	// get the next incoming header
	h, ok := <-headers
	if !ok {
		return nil, errors.Wrap(errors.ErrNetwork, "Subscription closed without returning any headers")
	}

	// A short delay so all queries on that block work as expected
	c.waitForTxIndex()
	return &h, nil
}

// WaitForHeight subscribes to headers and returns as soon as a header arrives
// equal to or greater than the given height. If the requested height is in the past,
// it will still wait for the next block to arrive
func (c *Client) WaitForHeight(ctx context.Context, height int64) (*Header, error) {
	// ensure we close subscription at function return
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()

	headers := make(chan Header, 2)
	err := c.SubscribeHeaders(cctx, headers)
	if err != nil {
		return nil, err
	}

	// read headers until we find desired height
	for h := range headers {
		if h.Height >= height {
			// A short delay so all queries on that block work as expected
			c.waitForTxIndex()
			return &h, nil
		}
	}
	return nil, errors.Wrapf(errors.ErrNetwork, "Subscription closed before height %d", height)
}

// waitForTxIndex waits until all tx in last blocked are properly indexed for the queries
// If you got a block header event, you need to wait a little bit untl you can search it
func (c *Client) waitForTxIndex() {
	time.Sleep(100 * time.Millisecond)
}
