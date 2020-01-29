package bnsd

import (
	"bytes"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/x/cash"
)

func TestBnsCashController(t *testing.T) {
	db := store.MemStore()
	migration.MustInitPkg(db, "cash")
	var (
		alice = weavetest.NewCondition().Address()
		bob   = weavetest.NewCondition().Address()
		burn  = weave.Address(bytes.Repeat([]byte{0}, weave.AddressLength))
	)

	ctrl := cash.NewController(cash.NewBucket())
	if err := ctrl.CoinMint(db, alice, coin.NewCoin(100, 0, "IOV")); err != nil {
		t.Fatalf("mint: %s", err)
	}

	bnsCtrl := BnsCashController(ctrl)

	if err := bnsCtrl.MoveCoins(db, alice, bob, coin.NewCoin(10, 0, "IOV")); err != nil {
		t.Fatalf("transfer from alice to bob: %s", err)
	}

	if funds, err := bnsCtrl.Balance(db, alice); err != nil {
		t.Fatalf("alice balance: %s", err)
	} else if !funds.Equals(coin.Coins{coin.NewCoinp(90, 0, "IOV")}) {
		t.Fatalf("want 90 IOV, got %v", funds)
	}

	if funds, err := bnsCtrl.Balance(db, bob); err != nil {
		t.Fatalf("bob balance: %s", err)
	} else if !funds.Equals(coin.Coins{coin.NewCoinp(10, 0, "IOV")}) {
		t.Fatalf("want 10 IOV, got %v", funds)
	}

	// Sending to a burn wallet must remove coins from the system.
	if err := bnsCtrl.MoveCoins(db, alice, burn, coin.NewCoin(30, 0, "IOV")); err != nil {
		t.Fatalf("transfer from alice to burn: %s", err)
	}
	if funds, err := bnsCtrl.Balance(db, alice); err != nil {
		t.Fatalf("alice balance: %s", err)
	} else if !funds.Equals(coin.Coins{coin.NewCoinp(60, 0, "IOV")}) {
		t.Fatalf("want 60 IOV, got %v", funds)
	}
	if funds, err := bnsCtrl.Balance(db, burn); err != nil {
		t.Fatalf("alice balance: %s", err)
	} else if !funds.Equals(coin.Coins{}) {
		t.Fatalf("want empty wallet, got %v", funds)
	}
}
