package cash

import (
	"testing"

	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

func wallet(t testing.TB, kv weave.KVStore, addr weave.Address) coin.Coins {
	t.Helper()

	bucket := NewBucket()
	res, err := bucket.Get(kv, addr)
	if err != nil {
		t.Fatalf("cannot get wallet for %q: %s", addr, err)
	}
	return AsCoins(res)
}

type issueCmd struct {
	addr    weave.Address
	amount  coin.Coin
	wantErr *errors.Error
}

type moveCmd struct {
	sender    weave.Address
	recipient weave.Address
	amount    coin.Coin
	wantErr   *errors.Error
}

type checkCmd struct {
	addr            weave.Address
	wantInWallet    []coin.Coin
	wantNotInWallet []coin.Coin
}

func TestIssueCoins(t *testing.T) {
	addr1 := weavetest.NewCondition().Address()
	addr2 := weavetest.NewCondition().Address()

	controller := NewController(NewBucket())

	plus := coin.NewCoin(500, 1000, "FOO")
	minus := coin.NewCoin(-400, -600, "FOO")
	total := coin.NewCoin(100, 400, "FOO")
	other := coin.NewCoin(1, 0, "DING")

	cases := map[string]struct {
		issue []issueCmd
		check []checkCmd
	}{
		"issue positive": {
			issue: []issueCmd{
				{addr: addr1, amount: plus},
			},
			check: []checkCmd{
				{addr: addr1, wantInWallet: []coin.Coin{plus, total}, wantNotInWallet: []coin.Coin{other}},
				{addr: addr2},
			},
		},
		"second issue negative": {
			issue: []issueCmd{
				{addr: addr1, amount: plus},
				{addr: addr1, amount: minus},
			},
			check: []checkCmd{
				{addr: addr1, wantInWallet: []coin.Coin{total}, wantNotInWallet: []coin.Coin{plus, other}},
				{addr: addr2},
			},
		},
		"issue to two chains": {
			issue: []issueCmd{
				{addr: addr1, amount: total},
				{addr: addr2, amount: other},
			},
			check: []checkCmd{
				{addr: addr1, wantInWallet: []coin.Coin{total}, wantNotInWallet: []coin.Coin{plus, other}},
				{addr: addr2, wantInWallet: []coin.Coin{other}, wantNotInWallet: []coin.Coin{plus, total}},
			},
		},
		"set back to zero": {
			issue: []issueCmd{
				{addr: addr2, amount: other},
				{addr: addr2, amount: other.Negative()},
			},
			check: []checkCmd{
				{addr: addr1},
				{addr: addr2},
			},
		},
		"set back to zero 2": {
			issue: []issueCmd{
				{addr: addr1, amount: total},
				{addr: addr1, amount: coin.NewCoin(coin.MaxInt, 0, "FOO"), wantErr: errors.ErrOverflow},
			},
			check: []checkCmd{
				{addr: addr1, wantInWallet: []coin.Coin{total}, wantNotInWallet: []coin.Coin{plus, other}},
				{addr: addr2},
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			kv := store.MemStore()
			migration.MustInitPkg(kv, "cash")

			for i, issue := range tc.issue {
				if err := controller.CoinMint(kv, issue.addr, issue.amount); !issue.wantErr.Is(err) {
					t.Fatalf("issue #%d: unexpected error: %+v", i, err)
				}
			}

			for i, check := range tc.check {
				w := wallet(t, kv, check.addr)

				if len(check.wantInWallet) == 0 && w != nil {
					t.Errorf("check #%d: expected an empty wallet: %#v", i, w)
				}

				for _, coin := range check.wantInWallet {
					if !w.Contains(coin) {
						t.Errorf("check #%d: missing coin in the wallet: %v", i, coin)
					}
				}
				for _, coin := range check.wantNotInWallet {
					if w.Contains(coin) {
						t.Errorf("check #%d: unwanted coin in the wallet: %v", i, coin)
					}
				}
			}
		})
	}

}

func TestMoveCoins(t *testing.T) {
	addr1 := weavetest.NewCondition().Address()
	addr2 := weavetest.NewCondition().Address()
	addr3 := weavetest.NewCondition().Address()

	controller := NewController(NewBucket())

	cc := "MONY"
	bank := coin.NewCoin(50000, 0, cc)
	send := coin.NewCoin(300, 0, cc)
	rem := coin.NewCoin(49700, 0, cc)

	cases := map[string]struct {
		issue issueCmd
		move  moveCmd
		check []checkCmd
	}{
		"cannot move money that you don't have": {
			issue: issueCmd{addr: addr3, amount: bank},
			move:  moveCmd{sender: addr1, recipient: addr2, amount: send, wantErr: errors.ErrEmpty},
			check: []checkCmd{
				{addr: addr2},
				{addr: addr3, wantInWallet: []coin.Coin{bank}},
			},
		},
		"simple send": {
			issue: issueCmd{addr: addr1, amount: bank},
			move:  moveCmd{sender: addr1, recipient: addr2, amount: send},
			check: []checkCmd{
				{addr: addr1, wantInWallet: []coin.Coin{rem}, wantNotInWallet: []coin.Coin{bank}},
				{addr: addr2, wantInWallet: []coin.Coin{send}, wantNotInWallet: []coin.Coin{bank}},
			},
		},
		"cannot send negative": {
			issue: issueCmd{addr: addr1, amount: bank},
			move:  moveCmd{sender: addr1, recipient: addr2, amount: send.Negative(), wantErr: errors.ErrAmount},
		},
		"cannot send more than you have": {
			issue: issueCmd{addr: addr1, amount: rem},
			move:  moveCmd{sender: addr1, recipient: addr2, amount: bank, wantErr: errors.ErrAmount},
		},
		"cannot send zero": {
			issue: issueCmd{addr: addr1, amount: bank},
			move:  moveCmd{sender: addr1, recipient: addr2, amount: coin.NewCoin(0, 0, cc), wantErr: errors.ErrAmount},
		},
		"cannot send wrong currency": {
			issue: issueCmd{addr: addr1, amount: bank},
			move:  moveCmd{sender: addr1, recipient: addr2, amount: coin.NewCoin(500, 0, "BAD"), wantErr: errors.ErrAmount},
		},
		"send everything": {
			issue: issueCmd{addr: addr1, amount: bank},
			move:  moveCmd{sender: addr1, recipient: addr2, amount: bank},
			check: []checkCmd{
				{addr: addr1, wantNotInWallet: []coin.Coin{bank}},
				{addr: addr2, wantInWallet: []coin.Coin{bank}},
			},
		},
		"send to self": {
			issue: issueCmd{addr: addr1, amount: rem},
			move:  moveCmd{sender: addr1, recipient: addr1, amount: send},
			check: []checkCmd{
				{addr: addr1, wantInWallet: []coin.Coin{send, rem}, wantNotInWallet: []coin.Coin{bank}},
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			kv := store.MemStore()
			migration.MustInitPkg(kv, "cash")

			if err := controller.CoinMint(kv, tc.issue.addr, tc.issue.amount); !tc.issue.wantErr.Is(err) {
				t.Fatalf("unexpected coin minting error: %+v", err)
			}

			if err := controller.MoveCoins(kv, tc.move.sender, tc.move.recipient, tc.move.amount); !tc.move.wantErr.Is(err) {
				t.Fatalf("unexpected coin transfer error: %+v", err)
			}

			for i, check := range tc.check {
				w := wallet(t, kv, check.addr)

				if len(check.wantInWallet) == 0 && w != nil {
					t.Errorf("check #%d: expected an empty wallet: %#v", i, w)
				}

				for _, coin := range check.wantInWallet {
					if !w.Contains(coin) {
						t.Errorf("check #%d: missing coin in the wallet: %v", i, coin)
					}
				}
				for _, coin := range check.wantNotInWallet {
					if w.Contains(coin) {
						t.Errorf("check #%d: unwanted coin in the wallet: %v", i, coin)
					}
				}
			}
		})
	}
}

func TestBalance(t *testing.T) {
	store := store.MemStore()
	migration.MustInitPkg(store, "cash")

	ctrl := NewController(NewBucket())

	addr1 := weavetest.NewCondition().Address()
	coin1 := coin.NewCoin(1, 20, "BTC")
	if err := ctrl.CoinMint(store, addr1, coin1); err != nil {
		t.Fatalf("cannot issue coins: %s", err)
	}

	addr2 := weavetest.NewCondition().Address()
	coin2_1 := coin.NewCoin(3, 40, "ETH")
	coin2_2 := coin.NewCoin(5, 0, "DOGE")
	if err := ctrl.CoinMint(store, addr2, coin2_1); err != nil {
		t.Fatalf("cannot issue coins: %s", err)
	}
	if err := ctrl.CoinMint(store, addr2, coin2_2); err != nil {
		t.Fatalf("cannot issue coins: %s", err)
	}

	cases := map[string]struct {
		addr      weave.Address
		wantCoins coin.Coins
		wantErr   *errors.Error
	}{
		"non existing account": {
			addr:    weavetest.NewCondition().Address(),
			wantErr: errors.ErrNotFound,
		},
		"existing account with one coin": {
			addr:      addr1,
			wantCoins: coin.Coins{&coin1},
		},
		"existing account with two coins": {
			addr: addr2,
			// Coins are stored in normalized form
			// https://github.com/iov-one/weave/pull/316#discussion_r256763396
			wantCoins: coin.Coins{&coin2_2, &coin2_1},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			coins, err := ctrl.Balance(store, tc.addr)
			if !tc.wantErr.Is(err) {
				t.Fatalf("want %q error, got %q", tc.wantErr, err)
			}
			if !tc.wantCoins.Equals(coins) {
				t.Logf("want %q", tc.wantCoins)
				t.Logf("got %q", coins)
				t.Fatal("unexpected coins amount")
			}
		})
	}
}
