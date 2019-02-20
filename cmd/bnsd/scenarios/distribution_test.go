package scenarios

import (
	"testing"

	"github.com/iov-one/weave"
	bnsdApp "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/distribution"
)

func TestRevenueDistribution(t *testing.T) {
	admin := client.GenPrivateKey()
	recipients := []*crypto.PrivateKey{
		client.GenPrivateKey(),
		client.GenPrivateKey(),
	}
	newRevenueTx := &bnsdApp.Tx{
		Sum: &bnsdApp.Tx_NewRevenueMsg{
			NewRevenueMsg: &distribution.NewRevenueMsg{
				Admin: admin.PublicKey().Address(),
				Recipients: []*distribution.Recipient{
					{Address: recipients[0].PublicKey().Address(), Weight: 1},
					{Address: recipients[1].PublicKey().Address(), Weight: 2},
				},
			},
		},
	}
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

	revenueAddress := distribution.RevenueAccount(revenueID)
	t.Logf("new revenue stream account: %s", revenueAddress)

	delayForRateLimits()

	// Now that we know what is the revenue stream account address we can
	// send coins there for later distribution.
	// Alice has plenty of money.
	sendCoinsTx := client.BuildSendTx(
		alice.PublicKey().Address(),
		revenueAddress,
		x.NewCoin(0, 7, "IOV"),
		"an income that is to be split using revenue distribution")
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

	delayForRateLimits()

	distributeTx := &bnsdApp.Tx{
		Sum: &bnsdApp.Tx_DistributeMsg{
			DistributeMsg: &distribution.DistributeMsg{
				RevenueID: revenueID,
			},
		},
	}
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

	// Revenue stream received funds from alice and its distribution was
	// requested. Funds should be split proportianally to their weights
	// between the recepients and moved to their accounts.
	// 7 IOV cents should be split between parties.
	assertWalletCoins(t, admin.PublicKey().Address(), 1)
	assertWalletCoins(t, recipients[0].PublicKey().Address(), 2)
	assertWalletCoins(t, recipients[1].PublicKey().Address(), 4)

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
		t.Fatal("wallet has no coins")
	}

	wantCoin := x.NewCoin(0, wantIOVCents, "IOV")
	gotCoins := x.Coins(w.Wallet.Coins)
	if !gotCoins.Equals(x.Coins{&wantCoin}) {
		t.Fatalf("wallet state: %s", gotCoins)
	}
}
