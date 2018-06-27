package utils

import (
	"testing"
	"time"

	"github.com/confio/weave/x"
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
	res := <-bcp.BroadcastTx(tx)
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
