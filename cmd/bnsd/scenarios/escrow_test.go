package scenarios

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/x/escrow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryEscrowExists(t *testing.T) {
	walletResp, err := bnsClient.GetWallet(escrowContract.Address())
	// then
	require.NoError(t, err)
	require.NotNil(t, walletResp)
	require.Len(t, walletResp.Wallet.Coins, 1)
	assert.True(t, walletResp.Wallet.Coins[0].Whole > 0)
}

func TestEscrowRelease(t *testing.T) {
	// query distribution accounts start balance
	walletResp, err := bnsClient.GetWallet(distrContractAddr)
	require.NoError(t, err)
	startBalance := coin.Coin{Ticker: "IOV"}
	if walletResp != nil {
		startBalance = *walletResp.Wallet.Coins[0]
	}
	aNonce := client.NewNonce(bnsClient, alice.PublicKey().Address())
	// when releasing 1 IOV by the arbiter
	_, _, escrowID, _ := escrowContract.Parse()

	releaseEscrowTX := &app.Tx{
		Sum: &app.Tx_ReleaseEscrowMsg{
			ReleaseEscrowMsg: &escrow.ReleaseEscrowMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: escrowID,
				Amount:   []*coin.Coin{{Ticker: "IOV", Whole: 1}},
			},
		},
	}
	releaseEscrowTX.Fee(alice.PublicKey().Address(), antiSpamFee)
	_, _, contractID, _ := multiSigContract.Parse()
	releaseEscrowTX.Multisig = [][]byte{contractID}

	seq, err := aNonce.Next()
	require.NoError(t, err)
	require.NoError(t, client.SignTx(releaseEscrowTX, alice, chainID, seq))
	resp := bnsClient.BroadcastTx(releaseEscrowTX)

	// then
	require.NoError(t, resp.IsError())

	// and check it was added to the distr account
	walletResp, err = bnsClient.GetWallet(distrContractAddr)
	require.NoError(t, err)
	require.NotNil(t, walletResp)
	require.True(t, walletResp.Height >= resp.Response.Height)
	require.True(t, len(walletResp.Wallet.Coins) == 1)
	// new balance should be higher
	assert.False(t, startBalance.IsGTE(*walletResp.Wallet.Coins[0]), "%s not > %s", *walletResp.Wallet.Coins[0], startBalance)
}
