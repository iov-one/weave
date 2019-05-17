package scenarios

import (
	"testing"

	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/cmd/bnsd/scenarios/bnsdtest"
	"github.com/iov-one/weave/coin"
	"github.com/stretchr/testify/require"
)

func TestSendTokenWithFee(t *testing.T) {
	env, cleanup := bnsdtest.StartBnsd(t)
	defer cleanup()

	emilia := client.GenPrivateKey()
	aNonce := client.NewNonce(env.Client, env.Alice.PublicKey().Address())

	walletResp, err := env.Client.GetWallet(env.Alice.PublicKey().Address())
	require.NoError(t, err)
	require.NotNil(t, walletResp, "address not found")
	require.NotEmpty(t, walletResp.Wallet.Coins)

	heights := make([]int64, len(walletResp.Wallet.Coins))
	for i, c := range walletResp.Wallet.Coins {
		// send a coin from Alice to Emilia
		cc := coin.Coin{
			Ticker:     c.Ticker,
			Fractional: 0,
			Whole:      1,
		}

		seq, err := aNonce.Next()
		require.NoError(t, err)
		tx := client.BuildSendTx(env.Alice.PublicKey().Address(), emilia.PublicKey().Address(), cc, "test tx with fee")
		tx.Fee(env.Alice.PublicKey().Address(), env.AntiSpamFee)
		require.NoError(t, client.SignTx(tx, env.Alice, env.ChainID, seq))
		resp := env.Client.BroadcastTx(tx)
		require.NoError(t, resp.IsError())
		heights[i] = resp.Response.Height
	}
	walletResp, err = env.Client.GetWallet(emilia.PublicKey().Address())
	require.NoError(t, err)
	t.Log("message", "done", "height", heights, "coins", walletResp.Wallet.Coins)
}

func TestQueryCurrencies(t *testing.T) {
	env, cleanup := bnsdtest.StartBnsd(t)
	defer cleanup()

	l, err := client.NewClient(env.Client.TendermintClient()).Currencies()
	require.NoError(t, err)
	require.NotNil(t, l, "no currencies found")
	require.True(t, len(l.Currencies) > 0)
}
