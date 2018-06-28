package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/rpc/client"
	rpctest "github.com/tendermint/tendermint/rpc/test"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/confio/weave/x"
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

	conn := client.NewLocal(node)
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
	conn := client.NewLocal(node)
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
	conn := client.NewLocal(node)
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
	conn := client.NewLocal(node)
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
	headers := make(chan interface{}, 4)

	conn := client.NewLocal(node)
	bcp := NewClient(conn)

	cancel, err := bcp.SubscribeHeaders(headers)
	require.NoError(t, err)

	// get two headers and cancel
	data := <-headers
	data2 := <-headers
	cancel()

	evt, ok := data.(tmtypes.EventDataNewBlockHeader)
	require.True(t, ok)
	evt2, ok := data2.(tmtypes.EventDataNewBlockHeader)
	require.True(t, ok)

	assert.NotNil(t, evt.Header)
	assert.NotNil(t, evt2.Header)
	assert.NotEmpty(t, evt.Header.ChainID)
	assert.NotEmpty(t, evt.Header.Height)
	assert.Equal(t, evt.Header.ChainID, evt2.Header.ChainID)
	assert.Equal(t, evt.Header.Height+1, evt2.Header.Height)

	// nothing else should be produced, let's wait 100ms to be sure
	timer := time.After(100 * time.Millisecond)
	select {
	case evt := <-headers:
		require.Nil(t, evt, "This must be nil from a closed channel")
	case <-timer:
		// we want this to fire
	}
}
