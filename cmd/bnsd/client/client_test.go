package client

import (
	"sync"
	"testing"
	"time"

	"github.com/iov-one/weave/coin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/rpc/client"
	rpctest "github.com/tendermint/tendermint/rpc/test"
)

// blocks go by fast, no need to wait seconds....
func fastWaiter(delta int64) (abort error) {
	delay := time.Duration(delta) * 5 * time.Millisecond
	time.Sleep(delay)
	return nil
}

var _ client.Waiter = fastWaiter

func TestMainSetup(t *testing.T) {
	config := rpctest.GetConfig()
	assert.Equal(t, "SetInTestMain", config.Moniker)

	conn := client.NewLocal(node)
	status, err := conn.Status()
	assert.Nil(t, err)
	assert.Equal(t, "SetInTestMain", status.NodeInfo.Moniker)

	// wait for some blocks to be produced....
	err = client.WaitForHeight(conn, 5, fastWaiter)
	assert.Nil(t, err)
	status, err = conn.Status()
	assert.Nil(t, err)
	assert.True(t, status.SyncInfo.LatestBlockHeight > 4)
}

func TestWalletQuery(t *testing.T) {
	conn := NewLocalConnection(node)
	bcp := NewClient(conn)
	err := client.WaitForHeight(conn, 5, fastWaiter)
	assert.Nil(t, err)

	// bad address returns error
	_, err = bcp.GetWallet([]byte{1, 2, 3, 4})
	assert.Error(t, err)

	// missing account returns nothing
	missing := GenPrivateKey().PublicKey().Address()
	wallet, err := bcp.GetWallet(missing)
	assert.Nil(t, err)
	assert.Nil(t, wallet)

	// genesis account returns something
	address := faucet.PublicKey().Address()
	wallet, err = bcp.GetWallet(address)
	assert.Nil(t, err)
	require.NotNil(t, wallet)
	// make sure we get some reasonable height
	assert.True(t, wallet.Height > 4)
	// ensure the key matches
	assert.EqualValues(t, address, wallet.Address)
	// check the wallet
	require.Equal(t, 1, len(wallet.Wallet.Coins))
	coin := wallet.Wallet.Coins[0]
	assert.Equal(t, initBalance.Whole, coin.Whole)
	assert.Equal(t, initBalance.Ticker, coin.Ticker)
}

func TestNonce(t *testing.T) {
	addr := GenPrivateKey().PublicKey().Address()
	conn := NewLocalConnection(node)
	bcp := NewClient(conn)

	nonce := NewNonce(bcp, addr)
	n, err := nonce.Next()
	assert.Nil(t, err)
	assert.Equal(t, int64(0), n)

	n, err = nonce.Next()
	assert.Nil(t, err)
	assert.Equal(t, int64(1), n)

	n, err = nonce.Next()
	assert.Nil(t, err)
	assert.Equal(t, int64(2), n)

	n, err = nonce.Query()
	assert.Nil(t, err)
	assert.Equal(t, int64(0), n)
}

func TestSendMoney(t *testing.T) {
	conn := NewLocalConnection(node)
	bcp := NewClient(conn)

	rcpt := GenPrivateKey().PublicKey().Address()
	src := faucet.PublicKey().Address()

	nonce := NewNonce(bcp, src)
	chainID := getChainID()

	// build the tx
	amount := coin.Coin{Whole: 1000, Ticker: initBalance.Ticker}
	tx := BuildSendTx(src, rcpt, amount, "Send 1")
	n, err := nonce.Query()
	assert.Nil(t, err)
	err = SignTx(tx, faucet, chainID, n)
	assert.Nil(t, err)

	// now post it
	res := bcp.BroadcastTx(tx)
	assert.Nil(t, res.IsError())

	// verify nonce incremented on chain
	n2, err := nonce.Query()
	assert.Nil(t, err)
	assert.Equal(t, n+1, n2)

	// verify wallet has cash
	wallet, err := bcp.GetWallet(rcpt)
	assert.Nil(t, err)
	require.NotNil(t, wallet)
	// check the wallet
	require.Equal(t, 1, len(wallet.Wallet.Coins))
	coin := wallet.Wallet.Coins[0]
	assert.Equal(t, int64(1000), coin.Whole)
	assert.Equal(t, initBalance.Ticker, coin.Ticker)
}

func TestSubscribeHeaders(t *testing.T) {
	conn := NewLocalConnection(node)
	bcp := NewClient(conn)

	headers := make(chan *Header, 4)
	cancel, err := bcp.SubscribeHeaders(headers)
	assert.Nil(t, err)

	// get two headers and cancel
	h := <-headers
	h2 := <-headers
	cancel()

	assert.NotNil(t, h)
	assert.NotNil(t, h2)
	assert.NotEmpty(t, h.ChainID)
	assert.NotEmpty(t, h.Height)
	assert.Equal(t, h.ChainID, h2.ChainID)
	assert.Equal(t, h.Height+1, h2.Height)

	// nothing else should be produced, let's wait 100ms to be sure
	timer := time.After(100 * time.Millisecond)
	select {
	case evt := <-headers:
		require.Nil(t, evt, "This must be nil from a closed channel")
	case <-timer:
		// we want this to fire
	}
}

func TestSendMultipleTx(t *testing.T) {
	conn := NewLocalConnection(node)
	bcp := NewClient(conn)

	friend := GenPrivateKey()
	rcpt := friend.PublicKey().Address()
	src := faucet.PublicKey().Address()

	nonce := NewNonce(bcp, src)
	chainID, err := bcp.ChainID()
	amount := coin.Coin{Whole: 1000, Ticker: initBalance.Ticker}
	assert.Nil(t, err)

	// a prep transaction, so the recipient has something to send
	prep := BuildSendTx(src, rcpt, amount, "Send 1")
	n, err := nonce.Next()
	assert.Nil(t, err)
	err = SignTx(prep, faucet, chainID, n)
	assert.Nil(t, err)

	// from sender with a different nonce
	tx := BuildSendTx(src, rcpt, amount, "Send 2")
	n, err = nonce.Next()
	assert.Nil(t, err)
	err = SignTx(tx, faucet, chainID, n)
	assert.Nil(t, err)

	// and a third one to return from rcpt to sender
	// nonce must be 0
	tx2 := BuildSendTx(rcpt, src, amount, "Return")
	err = SignTx(tx2, friend, chainID, 0)
	assert.Nil(t, err)

	// first, we send the one transaction so the next two will succeed
	prepResp := bcp.BroadcastTx(prep)
	assert.Nil(t, prepResp.IsError())
	prepH := prepResp.Response.Height

	txResp := make(chan BroadcastTxResponse, 2)
	headers, cancel, err := bcp.Subscribe(QueryNewBlockHeader)
	assert.Nil(t, err)

	// to avoid race conditions, wait for a new header
	// event, then immediately send off the two tx
	var ready, start sync.WaitGroup
	ready.Add(2)
	start.Add(1)

	go func() {
		ready.Done()
		start.Wait()
		bcp.BroadcastTxAsync(tx, txResp)
	}()
	go func() {
		ready.Done()
		start.Wait()
		bcp.BroadcastTxAsync(tx2, txResp)
	}()

	ready.Wait()
	<-headers
	start.Done()
	cancel()

	// both succeed
	resp := <-txResp
	resp2 := <-txResp
	assert.Nil(t, resp.IsError())
	assert.Nil(t, resp2.IsError())
	assert.True(t, resp.Response.Height > prepH+1)
	assert.True(t, resp2.Response.Height > prepH+1)
}
