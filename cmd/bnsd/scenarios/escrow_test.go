package scenarios

import (
	"testing"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/cmd/bnsd/scenarios/bnsdtest"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/escrow"
)

func TestQueryEscrowExists(t *testing.T) {
	env, cleanup := bnsdtest.StartBnsd(t)
	defer cleanup()

	walletResp, err := env.Client.GetWallet(env.EscrowContract.Address())
	// then
	assert.Nil(t, err)
	assert.Equal(t, true, walletResp != nil)
	assert.Equal(t, 1, len(walletResp.Wallet.Coins))
	assert.Equal(t, true, walletResp.Wallet.Coins[0].Whole > 0)
}

func TestEscrowRelease(t *testing.T) {
	env, cleanup := bnsdtest.StartBnsd(t)
	defer cleanup()

	// query distribution accounts start balance
	walletResp, err := env.Client.GetWallet(env.DistrContractAddr)
	assert.Nil(t, err)
	startBalance := coin.Coin{Ticker: "IOV"}
	if walletResp != nil {
		startBalance = *walletResp.Wallet.Coins[0]
	}
	aNonce := client.NewNonce(env.Client, env.Alice.PublicKey().Address())
	// when releasing 1 IOV by the arbiter
	_, _, escrowID, err := env.EscrowContract.Parse()
	if err != nil {
		t.Fatalf("cannot parse escrow contract: %s", err)
	}

	releaseEscrowTX := &bnsd.Tx{
		Sum: &bnsd.Tx_EscrowReleaseMsg{
			EscrowReleaseMsg: &escrow.ReleaseMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: escrowID,
				Amount:   []*coin.Coin{{Ticker: "IOV", Whole: 1}},
			},
		},
	}
	releaseEscrowTX.Fee(env.Alice.PublicKey().Address(), env.AntiSpamFee)
	_, _, contractID, _ := env.MultiSigContract.Parse()
	releaseEscrowTX.Multisig = [][]byte{contractID}

	seq, err := aNonce.Next()
	assert.Nil(t, err)
	assert.Nil(t, client.SignTx(releaseEscrowTX, env.Alice, env.ChainID, seq))
	resp := env.Client.BroadcastTx(releaseEscrowTX)

	// then
	assert.Nil(t, resp.IsError())

	// and check it was added to the distr account
	walletResp, err = env.Client.GetWallet(env.DistrContractAddr)
	assert.Nil(t, err)
	assert.Equal(t, true, walletResp != nil)
	assert.Equal(t, true, walletResp.Height >= resp.Response.Height)
	assert.Equal(t, true, len(walletResp.Wallet.Coins) == 1)
	// new balance should be higher
	assert.Equal(t, false, startBalance.IsGTE(*walletResp.Wallet.Coins[0]))
}
