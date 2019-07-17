package scenarios

import (
	"testing"

	"github.com/iov-one/weave"
	bnsdApp "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/cmd/bnsd/scenarios/bnsdtest"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/x/distribution"
)

func TestRevenueDistribution(t *testing.T) {
	env, cleanup := bnsdtest.StartBnsd(t,
		bnsdtest.WithMsgFee("distribution/newrevenue", coin.NewCoin(2, 0, "IOV")),
		bnsdtest.WithMsgFee("distribution/distribute", coin.NewCoin(0, 200000000, "IOV")),
		bnsdtest.WithMsgFee("distribution/resetRevenue", coin.NewCoin(1, 0, "IOV")),
	)
	defer cleanup()

	admin := client.GenPrivateKey()
	bnsdtest.SeedAccountWithTokens(t, env, admin.PublicKey().Address())

	destinations := []weave.Address{
		weavetest.NewKey().PublicKey().Address(),
		weavetest.NewKey().PublicKey().Address(),
	}
	newRevenueTx := &bnsdApp.Tx{
		Sum: &bnsdApp.Tx_DistributionCreateMsg{
			DistributionCreateMsg: &distribution.CreateMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Admin:    admin.PublicKey().Address(),
				Destinations: []*distribution.Destination{
					{Address: destinations[0], Weight: 1},
					{Address: destinations[1], Weight: 2},
				},
			},
		},
	}
	newRevenueTx.Fee(admin.PublicKey().Address(), coin.NewCoin(2, 0, "IOV"))

	adminNonce := client.NewNonce(env.Client, admin.PublicKey().Address())
	seq, err := adminNonce.Next()
	if err != nil {
		t.Fatalf("cannot acquire admin nonce sequence: %s", err)
	}
	if err := client.SignTx(newRevenueTx, admin, env.ChainID, seq); err != nil {
		t.Fatalf("cannot sing revenue creation transaction: %s", err)
	}
	resp := env.Client.BroadcastTx(newRevenueTx)
	if err := resp.IsError(); err != nil {
		t.Fatalf("cannot broadcast new revenue transaction: %s", err)
	}
	revenueID := weave.Address(resp.Response.DeliverTx.GetData())
	t.Logf("new revenue stream id: %s", revenueID)

	revenueAddress := distribution.RevenueAccount(revenueID)
	t.Logf("new revenue stream account: %s", revenueAddress)

	// Now that we know what is the revenue stream account address we can
	// send coins there for later distribution.
	// Alice has plenty of money.
	sendCoinsTx := client.BuildSendTx(
		env.Alice.PublicKey().Address(),
		revenueAddress,
		coin.NewCoin(0, 7, "IOV"),
		"an income that is to be split using revenue distribution")
	sendCoinsTx.Fee(env.Alice.PublicKey().Address(), env.AntiSpamFee)

	aliceNonce := client.NewNonce(env.Client, env.Alice.PublicKey().Address())
	seq, err = aliceNonce.Next()
	if err != nil {
		t.Fatalf("cannot acquire alice nonce sequence: %s", err)
	}
	if err := client.SignTx(sendCoinsTx, env.Alice, env.ChainID, seq); err != nil {
		t.Fatalf("alice cannot sign coin transfer transaction: %s", err)
	}
	resp = env.Client.BroadcastTx(sendCoinsTx)
	if err := resp.IsError(); err != nil {
		t.Fatalf("cannot broadcast coin sending transaction from alice: %s", err)
	}
	t.Logf("alice transferred funds to revenue %s account: %s", revenueID, string(resp.Response.DeliverTx.GetData()))
	assertWalletCoins(t, env, revenueAddress, 7)

	// Revenue reset must distribute the funds before changing the
	// configuration.

	resetRevenueTx := &bnsdApp.Tx{
		Sum: &bnsdApp.Tx_DistributionResetMsg{
			DistributionResetMsg: &distribution.ResetMsg{
				Metadata:  &weave.Metadata{Schema: 1},
				RevenueID: revenueID,
				Destinations: []*distribution.Destination{
					{Address: destinations[0], Weight: 321},
				},
			},
		},
	}
	resetRevenueTx.Fee(admin.PublicKey().Address(), coin.NewCoin(1, 0, "IOV"))
	seq, err = adminNonce.Next()
	if err != nil {
		t.Fatalf("cannot acquire admin nonce sequence: %s", err)
	}
	if err := client.SignTx(resetRevenueTx, admin, env.ChainID, seq); err != nil {
		t.Fatalf("cannot sing revenue distribution transaction: %s", err)
	}
	if err := env.Client.BroadcastTx(resetRevenueTx).IsError(); err != nil {
		t.Fatalf("cannot broadcast revenue distribution transaction: %s", err)
	}

	// Revenue stream received funds from alice and its distribution was
	// requested. Funds should be split proportionally to their weights
	// between the destinations and moved to their accounts.
	// 7 IOV cents should be split between parties.
	assertWalletCoins(t, env, revenueAddress, 1)
	assertWalletCoins(t, env, destinations[0], 2)
	assertWalletCoins(t, env, destinations[1], 4)

	// Send more coins to the revenue account.
	sendCoinsTx = client.BuildSendTx(
		env.Alice.PublicKey().Address(),
		revenueAddress,
		coin.NewCoin(0, 11, "IOV"),
		"an income that is to be split using revenue distribution (2)")
	sendCoinsTx.Fee(env.Alice.PublicKey().Address(), env.AntiSpamFee)

	seq, err = aliceNonce.Next()
	if err != nil {
		t.Fatalf("cannot acquire alice nonce sequence: %s", err)
	}
	if err := client.SignTx(sendCoinsTx, env.Alice, env.ChainID, seq); err != nil {
		t.Fatalf("alice cannot sign coin transfer transaction: %s", err)
	}
	resp = env.Client.BroadcastTx(sendCoinsTx)
	if err := resp.IsError(); err != nil {
		t.Fatalf("cannot broadcast coin sending transaction from alice: %s", err)
	}
	t.Logf("alice transferred funds to revenue %s account: %s", revenueID, string(resp.Response.DeliverTx.GetData()))
	assertWalletCoins(t, env, revenueAddress, 12) // 11 + 1 leftover

	distributeTx := &bnsdApp.Tx{
		Sum: &bnsdApp.Tx_DistributionMsg{
			DistributionMsg: &distribution.DistributeMsg{
				Metadata:  &weave.Metadata{Schema: 1},
				RevenueID: revenueID,
			},
		},
	}
	distributeTx.Fee(admin.PublicKey().Address(), coin.NewCoin(0, 200000000, "IOV"))

	seq, err = adminNonce.Next()
	if err != nil {
		t.Fatalf("cannot acquire admin nonce sequence: %s", err)
	}
	if err := client.SignTx(distributeTx, admin, env.ChainID, seq); err != nil {
		t.Fatalf("cannot sing revenue distribution transaction: %s", err)
	}
	if err := env.Client.BroadcastTx(distributeTx).IsError(); err != nil {
		t.Fatalf("cannot broadcast revenue distribution transaction: %s", err)
	}

	assertWalletCoins(t, env, revenueAddress, 0)
	assertWalletCoins(t, env, destinations[0], 14)
	assertWalletCoins(t, env, destinations[1], 4)
}

func assertWalletCoins(t *testing.T, env *bnsdtest.EnvConf, account weave.Address, wantIOVCents int64) {
	t.Helper()

	w, err := env.Client.GetWallet(account)
	if err != nil {
		t.Fatalf("cannot get first destinations wallet: %s", err)
	}
	if w == nil {
		t.Fatal("no wallet response") // ?!
	}
	if len(w.Wallet.Coins) == 0 {
		if wantIOVCents == 0 {
			return
		}
		t.Fatal("wallet has no coins")
	}

	wantCoin := coin.NewCoin(0, wantIOVCents, "IOV")
	gotCoins := coin.Coins(w.Wallet.Coins)
	if !gotCoins.Equals(coin.Coins{&wantCoin}) {
		t.Fatalf("wallet state: %s", gotCoins)
	}
}
