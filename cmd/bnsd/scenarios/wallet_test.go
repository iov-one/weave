package scenarios

import (
	"testing"

	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/cmd/bnsd/scenarios/bnsdtest"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestSendTokenWithFee(t *testing.T) {
	env, cleanup := bnsdtest.StartBnsd(t)
	defer cleanup()

	emilia := client.GenPrivateKey()
	aNonce := client.NewNonce(env.Client, env.Alice.PublicKey().Address())

	walletResp, err := env.Client.GetWallet(env.Alice.PublicKey().Address())
	assert.Nil(t, err)
	assert.Equal(t, true, walletResp != nil)
	assert.Equal(t, true, len(walletResp.Wallet.Coins) > 0)

	heights := make([]int64, len(walletResp.Wallet.Coins))
	for i, c := range walletResp.Wallet.Coins {
		// send a coin from Alice to Emilia
		cc := coin.Coin{
			Ticker:     c.Ticker,
			Fractional: 0,
			Whole:      1,
		}

		seq, err := aNonce.Next()
		assert.Nil(t, err)
		tx := client.BuildSendTx(env.Alice.PublicKey().Address(), emilia.PublicKey().Address(), cc, "test tx with fee")
		tx.Fee(env.Alice.PublicKey().Address(), env.AntiSpamFee)
		assert.Nil(t, client.SignTx(tx, env.Alice, env.ChainID, seq))
		resp := env.Client.BroadcastTx(tx)
		assert.Nil(t, resp.IsError())
		heights[i] = resp.Response.Height
	}
	walletResp, err = env.Client.GetWallet(emilia.PublicKey().Address())
	assert.Nil(t, err)
	t.Log("message", "done", "height", heights, "coins", walletResp.Wallet.Coins)
}

func TestQueryCurrencies(t *testing.T) {
	env, cleanup := bnsdtest.StartBnsd(t)
	defer cleanup()

	l, err := client.NewClient(env.Client.TendermintClient()).Currencies()
	assert.Nil(t, err)
	assert.Equal(t, true, len(l.Currencies) > 0)
}
