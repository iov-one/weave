package scenarios

import (
	"testing"

	"github.com/iov-one/weave"
	bnsdApp "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/x/distribution"
)

func TestRevenueDistribution(t *testing.T) {
	admin := client.GenPrivateKey()
	seedAccountWithTokens(admin.PublicKey().Address())

	recipients := []weave.Address{
		client.GenPrivateKey().PublicKey().Address(),
		client.GenPrivateKey().PublicKey().Address(),
	}
	newRevenueTx := &bnsdApp.Tx{
		Sum: &bnsdApp.Tx_NewRevenueMsg{
			NewRevenueMsg: &distribution.NewRevenueMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Admin:    admin.PublicKey().Address(),
				Recipients: []*distribution.Recipient{
					{Address: recipients[0], Weight: 1},
					{Address: recipients[1], Weight: 2},
				},
			},
		},
	}
	newRevenueTx.Fee(admin.PublicKey().Address(), coin.NewCoin(2, 0, "IOV"))

	adminNonce := client.NewNonce(bnsClient, admin.PublicKey().Address())
	seq, err := adminNonce.Next()
	if err != nil {
		t.Fatalf("cannot acquire admin nonce sequence: %s", err)
	}
	if err := client.SignTx(newRevenueTx, admin, chainID, seq); err != nil {
		t.Fatalf("cannot sing revenue creation transaction: %s", err)
	}
	resp := bnsClient.BroadcastTx(newRevenueTx)
	if err := resp.IsError(); err != nil {
		t.Fatalf("cannot broadcast new revenue transaction: %s", err)
	}
	revenueID := weave.Address(resp.Response.DeliverTx.GetData())
	t.Logf("new revenue stream id: %s", revenueID)

	revenueAddress, err := distribution.RevenueAccount(revenueID)
	if err != nil {
		t.Fatalf("cannot create a revenue account for %d: %s", revenueID, err)
	}
	t.Logf("new revenue stream account: %s", revenueAddress)

	delayForRateLimits()

	// Now that we know what is the revenue stream account address we can
	// send coins there for later distribution.
	// Alice has plenty of money.
	sendCoinsTx := client.BuildSendTx(
		alice.PublicKey().Address(),
		revenueAddress,
		coin.NewCoin(0, 7, "IOV"),
		"an income that is to be split using revenue distribution")
	sendCoinsTx.Fee(alice.PublicKey().Address(), antiSpamFee)

	aliceNonce := client.NewNonce(bnsClient, alice.PublicKey().Address())
	seq, err = aliceNonce.Next()
	if err != nil {
		t.Fatalf("cannot acquire alice nonce sequence: %s", err)
	}
	if err := client.SignTx(sendCoinsTx, alice, chainID, seq); err != nil {
		t.Fatalf("alice cannot sign coin transfer transaction: %s", err)
	}
	resp = bnsClient.BroadcastTx(sendCoinsTx)
	if err := resp.IsError(); err != nil {
		t.Fatalf("cannot broadcast coin sending transaction from alice: %s", err)
	}
	t.Logf("alice transferred funds to revenue %s account: %s", revenueID, string(resp.Response.DeliverTx.GetData()))
	assertWalletCoins(t, revenueAddress, 7)

	delayForRateLimits()

	// Revenue reset must distribute the funds before changing the
	// configuration.

	resetRevenueTx := &bnsdApp.Tx{
		Sum: &bnsdApp.Tx_ResetRevenueMsg{
			ResetRevenueMsg: &distribution.ResetRevenueMsg{
				Metadata:  &weave.Metadata{Schema: 1},
				RevenueID: revenueID,
				Recipients: []*distribution.Recipient{
					{Address: recipients[0], Weight: 321},
				},
			},
		},
	}
	resetRevenueTx.Fee(admin.PublicKey().Address(), coin.NewCoin(1, 0, "IOV"))
	seq, err = adminNonce.Next()
	if err != nil {
		t.Fatalf("cannot acquire admin nonce sequence: %s", err)
	}
	if err := client.SignTx(resetRevenueTx, admin, chainID, seq); err != nil {
		t.Fatalf("cannot sing revenue distribution transaction: %s", err)
	}
	if err := bnsClient.BroadcastTx(resetRevenueTx).IsError(); err != nil {
		t.Fatalf("cannot broadcast revenue distribution transaction: %s", err)
	}

	// Revenue stream received funds from alice and its distribution was
	// requested. Funds should be split proportianally to their weights
	// between the recepients and moved to their accounts.
	// 7 IOV cents should be split between parties.
	assertWalletCoins(t, revenueAddress, 1)
	assertWalletCoins(t, recipients[0], 2)
	assertWalletCoins(t, recipients[1], 4)

	// Send more coins to the revenue account.
	sendCoinsTx = client.BuildSendTx(
		alice.PublicKey().Address(),
		revenueAddress,
		coin.NewCoin(0, 11, "IOV"),
		"an income that is to be split using revenue distribution (2)")
	sendCoinsTx.Fee(alice.PublicKey().Address(), antiSpamFee)

	seq, err = aliceNonce.Next()
	if err != nil {
		t.Fatalf("cannot acquire alice nonce sequence: %s", err)
	}
	if err := client.SignTx(sendCoinsTx, alice, chainID, seq); err != nil {
		t.Fatalf("alice cannot sign coin transfer transaction: %s", err)
	}
	resp = bnsClient.BroadcastTx(sendCoinsTx)
	if err := resp.IsError(); err != nil {
		t.Fatalf("cannot broadcast coin sending transaction from alice: %s", err)
	}
	t.Logf("alice transferred funds to revenue %s account: %s", revenueID, string(resp.Response.DeliverTx.GetData()))
	assertWalletCoins(t, revenueAddress, 12) // 11 + 1 leftover

	distributeTx := &bnsdApp.Tx{
		Sum: &bnsdApp.Tx_DistributeMsg{
			DistributeMsg: &distribution.DistributeMsg{
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
	if err := client.SignTx(distributeTx, admin, chainID, seq); err != nil {
		t.Fatalf("cannot sing revenue distribution transaction: %s", err)
	}
	if err := bnsClient.BroadcastTx(distributeTx).IsError(); err != nil {
		t.Fatalf("cannot broadcast revenue distribution transaction: %s", err)
	}

	delayForRateLimits()

	assertWalletCoins(t, revenueAddress, 0)
	assertWalletCoins(t, recipients[0], 14)
	assertWalletCoins(t, recipients[1], 4)
}

func assertWalletCoins(t *testing.T, account weave.Address, wantIOVCents int64) {
	t.Helper()

	w, err := bnsClient.GetWallet(account)
	if err != nil {
		t.Fatalf("cannot get first recipients wallet: %s", err)
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
