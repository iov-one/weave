package client

import (
	"sync"
	"testing"
	"time"

	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/weavetest/assert"
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
	client.WaitForHeight(conn, 5, fastWaiter)
	status, err = conn.Status()
	assert.Nil(t, err)
	assert.Equal(t, true, status.SyncInfo.LatestBlockHeight > 4)
}

func TestWalletQuery(t *testing.T) {
	conn := NewLocalConnection(node)
	bcp := NewClient(conn)
	client.WaitForHeight(conn, 5, fastWaiter)

	// bad address returns error
	_, err := bcp.GetWallet([]byte{1, 2, 3, 4})
	assert.Equal(t, true, err != nil)

	// missing account returns nothing
	missing := GenPrivateKey().PublicKey().Address()
	wallet, err := bcp.GetWallet(missing)
	assert.Nil(t, err)
	assert.Nil(t, wallet)

	// genesis account returns something
	address := faucet.PublicKey().Address()
	wallet, err = bcp.GetWallet(address)
	assert.Nil(t, err)
	assert.Equal(t, true, wallet != nil)
	// make sure we get some reasonable height
	assert.Equal(t, true, wallet.Height > 4)
	// ensure the key matches
	assert.Equal(t, address, wallet.Address)
	// check the wallet
	assert.Equal(t, 1, len(wallet.Wallet.Coins))
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
	SignTx(tx, faucet, chainID, n)

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
	assert.Equal(t, true, wallet != nil)
	// check the wallet
	assert.Equal(t, 1, len(wallet.Wallet.Coins))
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

	assert.Equal(t, true, h != nil)
	assert.Equal(t, true, h2 != nil)
	assert.Equal(t, true, len(h.ChainID) > 0)
	assert.Equal(t, true, h.Height != 0)
	assert.Equal(t, h.ChainID, h2.ChainID)
	assert.Equal(t, h.Height+1, h2.Height)

	// nothing else should be produced, let's wait 100ms to be sure
	timer := time.After(100 * time.Millisecond)
	select {
	case evt := <-headers:
		assert.Nil(t, evt)
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
	SignTx(prep, faucet, chainID, n)

	// from sender with a different nonce
	tx := BuildSendTx(src, rcpt, amount, "Send 2")
	n, err = nonce.Next()
	assert.Nil(t, err)
	SignTx(tx, faucet, chainID, n)

	// and a third one to return from rcpt to sender
	// nonce must be 0
	tx2 := BuildSendTx(rcpt, src, amount, "Return")
	SignTx(tx2, friend, chainID, 0)

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
	assert.Equal(t, true, resp.Response.Height > prepH+1)
	assert.Equal(t, true, resp2.Response.Height > prepH+1)
}
