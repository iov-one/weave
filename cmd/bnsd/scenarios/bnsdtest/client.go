package bnsdtest

import (
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/errors"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
)

func throttle(c client.Client, frequency time.Duration) *ThrottledClient {
	// If client is already throttled, overwrite with the new frequency.
	if tc, ok := c.(*ThrottledClient); ok {
		c = tc.cli
	}

	return &ThrottledClient{
		cli:  c,
		stop: make(chan struct{}),
		tick: time.NewTicker(frequency),
	}
}

// ThrottleClient implements bnsd Client interface. All operations are
// throttled and executed at most with given frequency to avoid server
// throttling. Throttling is transparent for the client as all method calls are
// blocking.
type ThrottledClient struct {
	cli  client.Client
	stop chan struct{}
	tick *time.Ticker
}

var _ client.Client = (*ThrottledClient)(nil)

// Close free resources. Closing twice will cause panic.
func (t *ThrottledClient) Close() {
	close(t.stop)
	t.tick.Stop()
}

func (t *ThrottledClient) wait() error {
	select {
	case <-t.stop:
		return errors.Wrap(errors.ErrState, "closed client")
	case <-t.tick.C:
		return nil
	}
}
func (t *ThrottledClient) TendermintClient() rpcclient.Client {
	return t.cli.TendermintClient()
}

func (t *ThrottledClient) GetUser(addr weave.Address) (*client.UserResponse, error) {
	if err := t.wait(); err != nil {
		return nil, err
	}
	return t.cli.GetUser(addr)
}

func (t *ThrottledClient) GetWallet(addr weave.Address) (*client.WalletResponse, error) {
	if err := t.wait(); err != nil {
		return nil, err
	}
	return t.cli.GetWallet(addr)
}

func (t *ThrottledClient) BroadcastTx(tx weave.Tx) client.BroadcastTxResponse {
	if err := t.wait(); err != nil {
		return client.BroadcastTxResponse{Error: err}
	}
	return t.cli.BroadcastTx(tx)
}

func (t *ThrottledClient) BroadcastTxAsync(tx weave.Tx, out chan<- client.BroadcastTxResponse) {
	if err := t.wait(); err != nil {
		out <- client.BroadcastTxResponse{Error: err}
	}
	t.cli.BroadcastTxAsync(tx, out)
}

func (t *ThrottledClient) BroadcastTxSync(tx weave.Tx, timeout time.Duration) client.BroadcastTxResponse {
	if err := t.wait(); err != nil {
		return client.BroadcastTxResponse{Error: err}
	}
	return t.cli.BroadcastTxSync(tx, timeout)
}

func (t *ThrottledClient) AbciQuery(path string, data []byte) (client.AbciResponse, error) {
	return t.cli.AbciQuery(path, data)
}
