package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/rpc/client"
	rpctest "github.com/tendermint/tendermint/rpc/test"

	"github.com/iov-one/weave/x"
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
	require.NoError(t, err)
	assert.Equal(t, "SetInTestMain", status.NodeInfo.Moniker)

	// wait for some blocks to be produced....
	client.WaitForHeight(conn, 5, fastWaiter)
	status, err = conn.Status()
	require.NoError(t, err)
	assert.True(t, status.SyncInfo.LatestBlockHeight > 4)
}

func TestWalletQuery(t *testing.T) {
	missing := GenPrivateKey().PublicKey().Address()

	conn := NewLocalConnection(node)
	bcp := NewClient(conn)
	client.WaitForHeight(conn, 5, fastWaiter)

	// bad address returns error
	_, err := bcp.GetWallet([]byte{1, 2, 3, 4})
	assert.Error(t, err)

	// missing account returns nothing
	wallet, err := bcp.GetWallet(missing)
	assert.NoError(t, err)
	assert.Nil(t, wallet)

	// genesis account returns something
	money := faucet.PublicKey().Address()
	wallet, err = bcp.GetWallet(money)
	assert.NoError(t, err)
	require.NotNil(t, wallet)
	// make sure we get some reasonable height
	assert.True(t, wallet.Height > 4)
	// ensure the key matches
	assert.EqualValues(t, money, wallet.Address)
	// check the wallet
	assert.Equal(t, "faucet", wallet.Wallet.Name)
	require.Equal(t, 1, len(wallet.Wallet.Coins))
	coin := wallet.Wallet.Coins[0]
	assert.Equal(t, initBalance.Whole, coin.Whole)
	assert.Equal(t, initBalance.Ticker, coin.Ticker)
}

func TestWalletNameQuery(t *testing.T) {
	conn := NewLocalConnection(node)
	bcp := NewClient(conn)
	client.WaitForHeight(conn, 5, fastWaiter)

	// missing account returns nothing
	wallet, err := bcp.GetWalletByName("nobody")
	assert.NoError(t, err)
	assert.Nil(t, wallet)

	// genesis account returns something
	wallet, err = bcp.GetWalletByName("faucet")
	assert.NoError(t, err)
	require.NotNil(t, wallet)
	// make sure we get some reasonable height
	assert.True(t, wallet.Height > 4)
	// ensure the key matches
	assert.EqualValues(t, faucet.PublicKey().Address(),
		wallet.Address)
	// check the wallet
	assert.Equal(t, "faucet", wallet.Wallet.Name)
	require.Equal(t, 1, len(wallet.Wallet.Coins))
	coin := wallet.Wallet.Coins[0]
	// this may be reduced by other tests so no guarantees
	assert.True(t, coin.Whole > 100000)
	assert.Equal(t, initBalance.Ticker, coin.Ticker)
}

func TestNonce(t *testing.T) {
	addr := GenPrivateKey().PublicKey().Address()
	conn := NewLocalConnection(node)
	bcp := NewClient(conn)

	nonce := NewNonce(bcp, addr)
	n, err := nonce.Next()
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)

	n, err = nonce.Next()
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	n, err = nonce.Next()
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)

	n, err = nonce.Query()
	require.NoError(t, err)
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
	amount := x.Coin{Whole: 1000, Ticker: initBalance.Ticker}
	tx := BuildSendTx(src, rcpt, amount, "Send 1")
	n, err := nonce.Query()
	require.NoError(t, err)
	SignTx(tx, faucet, chainID, n)

	// now post it
	res := bcp.BroadcastTx(tx)
	require.NoError(t, res.IsError())

	// verify nonce incremented on chain
	n2, err := nonce.Query()
	require.NoError(t, err)
	assert.Equal(t, n+1, n2)

	// verify wallet has cash
	wallet, err := bcp.GetWallet(rcpt)
	assert.NoError(t, err)
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
	require.NoError(t, err)

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
	amount := x.Coin{Whole: 1000, Ticker: initBalance.Ticker}
	require.NoError(t, err)

	// a prep transaction, so the recipient has something to send
	prep := BuildSendTx(src, rcpt, amount, "Send 1")
	n, err := nonce.Next()
	require.NoError(t, err)
	SignTx(prep, faucet, chainID, n)

	// from sender with a different nonce
	tx := BuildSendTx(src, rcpt, amount, "Send 2")
	n, err = nonce.Next()
	require.NoError(t, err)
	SignTx(tx, faucet, chainID, n)

	// and a third one to return from rcpt to sender
	// nonce must be 0
	tx2 := BuildSendTx(rcpt, src, amount, "Return")
	SignTx(tx2, friend, chainID, 0)

	// first, we send the one transaction so the next two will succeed
	prepResp := bcp.BroadcastTx(prep)
	require.NoError(t, prepResp.IsError())
	prepH := prepResp.Response.Height

	txResp := make(chan BroadcastTxResponse, 2)
	headers := make(chan interface{}, 1)
	cancel, err := bcp.Subscribe(QueryNewBlockHeader, headers)
	require.NoError(t, err)

	// to avoid race conditions, wait for a new header
	// event, then immeidately send off the two tx
	<-headers
	go bcp.BroadcastTxAsync(tx, txResp)
	go bcp.BroadcastTxAsync(tx2, txResp)
	cancel()

	// both succeed and are in the same block
	resp := <-txResp
	resp2 := <-txResp
	assert.NoError(t, resp.IsError())
	assert.NoError(t, resp2.IsError())
	assert.Equal(t, resp.Response.Height, resp2.Response.Height)
	assert.True(t, resp.Response.Height > prepH+1)
}
