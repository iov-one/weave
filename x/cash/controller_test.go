package cash

import (
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getWallet(kv weave.KVStore, addr weave.Address) coin.Coins {
	bucket := NewBucket()
	res, err := bucket.Get(kv, addr)
	if err != nil {
		panic(err) // testing only
	}
	return AsCoins(res)
}

type issueCmd struct {
	addr   weave.Address
	amount coin.Coin
	isErr  bool
}

type moveCmd struct {
	sender weave.Address
	rcpt   weave.Address
	amount coin.Coin
	isErr  bool
}

type checkCmd struct {
	addr       weave.Address
	isNil      bool
	contains   []coin.Coin
	notContain []coin.Coin
}

func TestIssueCoins(t *testing.T) {
	addr1 := weavetest.NewCondition().Address()
	addr2 := weavetest.NewCondition().Address()

	controller := NewController(NewBucket())

	plus := coin.NewCoin(500, 1000, "FOO")
	minus := coin.NewCoin(-400, -600, "FOO")
	total := coin.NewCoin(100, 400, "FOO")
	other := coin.NewCoin(1, 0, "DING")

	cases := []struct {
		issue []issueCmd
		check []checkCmd
	}{
		// issue positive
		{
			issue: []issueCmd{{addr1, plus, false}},
			check: []checkCmd{
				{addr1, false, []coin.Coin{plus, total}, []coin.Coin{other}},
				{addr2, true, nil, nil},
			},
		},
		// second issue negative
		{
			issue: []issueCmd{{addr1, plus, false}, {addr1, minus, false}},
			check: []checkCmd{
				{addr1, false, []coin.Coin{total}, []coin.Coin{plus, other}},
				{addr2, true, nil, nil},
			},
		},
		// issue to two chains
		{
			issue: []issueCmd{{addr1, total, false}, {addr2, other, false}},
			check: []checkCmd{
				{addr1, false, []coin.Coin{total}, []coin.Coin{plus, other}},
				{addr2, false, []coin.Coin{other}, []coin.Coin{plus, total}},
			},
		},
		// set back to zero
		{
			issue: []issueCmd{{addr2, other, false}, {addr2, other.Negative(), false}},
			check: []checkCmd{
				{addr1, true, nil, nil},
				{addr2, true, nil, nil},
			},
		},
		// set back to zero
		{
			issue: []issueCmd{
				{addr1, total, false},
				{addr1, coin.NewCoin(coin.MaxInt, 0, "FOO"), true}},
			check: []checkCmd{
				{addr1, false, []coin.Coin{total}, []coin.Coin{plus, other}},
				{addr2, true, nil, nil},
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			kv := store.MemStore()

			for j, issue := range tc.issue {
				err := controller.IssueCoins(kv, issue.addr, issue.amount)
				if issue.isErr {
					require.Error(t, err, "%d", j)
				} else {
					require.NoError(t, err, "%d", j)
				}
			}

			for j, check := range tc.check {
				w := getWallet(kv, check.addr)
				if check.isNil {
					require.Nil(t, w, "%d", j)
				} else {
					require.NotNil(t, w, "%d", j)
					for k, has := range check.contains {
						assert.True(t, w.Contains(has), "%d/%d: %#v", j, k, w)
					}
					for k, not := range check.notContain {
						assert.False(t, w.Contains(not), "%d/%d: %#v", j, k, w)
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

	cases := []struct {
		issue issueCmd
		move  moveCmd
		check []checkCmd
	}{
		// cannot move money that you don't have
		{
			issue: issueCmd{addr3, bank, false},
			move:  moveCmd{addr1, addr2, send, true},
			check: []checkCmd{
				{addr2, true, nil, nil},
				{addr3, false, []coin.Coin{bank}, nil},
			},
		},
		// simple send
		{
			issue: issueCmd{addr1, bank, false},
			move:  moveCmd{addr1, addr2, send, false},
			check: []checkCmd{
				{addr1, false, []coin.Coin{rem}, []coin.Coin{bank}},
				{addr2, false, []coin.Coin{send}, []coin.Coin{bank}},
			},
		},
		// cannot send negative
		{
			issue: issueCmd{addr1, bank, false},
			move:  moveCmd{addr1, addr2, send.Negative(), true},
			check: nil,
		},
		// cannot send more than you have
		{
			issue: issueCmd{addr1, rem, false},
			move:  moveCmd{addr1, addr2, bank, true},
			check: nil,
		},
		// cannot send zero
		{
			issue: issueCmd{addr1, bank, false},
			move:  moveCmd{addr1, addr2, coin.NewCoin(0, 0, cc), true},
			check: nil,
		},
		// cannot send wrong currency
		{
			issue: issueCmd{addr1, bank, false},
			move:  moveCmd{addr1, addr2, coin.NewCoin(500, 0, "BAD"), true},
			check: nil,
		},
		// send everything
		{
			issue: issueCmd{addr1, bank, false},
			move:  moveCmd{addr1, addr2, bank, false},
			check: []checkCmd{
				{addr1, true, nil, nil},
				{addr2, false, []coin.Coin{bank}, nil},
			},
		},
		// send to self
		{
			issue: issueCmd{addr1, rem, false},
			move:  moveCmd{addr1, addr1, send, false},
			check: []checkCmd{
				{addr1, false, []coin.Coin{send, rem}, []coin.Coin{bank}},
			},
		},
		// TODO: check overflow
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			kv := store.MemStore()

			err := controller.IssueCoins(kv, tc.issue.addr, tc.issue.amount)
			if tc.issue.isErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			err = controller.MoveCoins(kv, tc.move.sender, tc.move.rcpt, tc.move.amount)
			if tc.move.isErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			for j, check := range tc.check {
				w := getWallet(kv, check.addr)
				if check.isNil {
					require.Nil(t, w, "%d", j)
				} else {
					require.NotNil(t, w, "%d", j)
					for k, has := range check.contains {
						assert.True(t, w.Contains(has), "%d/%d: %#v", j, k, w)
					}
					for k, not := range check.notContain {
						assert.False(t, w.Contains(not), "%d/%d: %#v", j, k, w)
					}
				}
			}
		})
	}
}

func TestBalance(t *testing.T) {
	store := store.MemStore()
	ctrl := NewController(NewBucket())

	addr1 := weavetest.NewCondition().Address()
	coin1 := coin.NewCoin(1, 20, "BTC")
	if err := ctrl.IssueCoins(store, addr1, coin1); err != nil {
		t.Fatalf("cannot issue coins: %s", err)
	}

	addr2 := weavetest.NewCondition().Address()
	coin2_1 := coin.NewCoin(3, 40, "ETH")
	coin2_2 := coin.NewCoin(5, 0, "DOGE")
	if err := ctrl.IssueCoins(store, addr2, coin2_1); err != nil {
		t.Fatalf("cannot issue coins: %s", err)
	}
	if err := ctrl.IssueCoins(store, addr2, coin2_2); err != nil {
		t.Fatalf("cannot issue coins: %s", err)
	}

	cases := map[string]struct {
		addr      weave.Address
		wantCoins coin.Coins
		wantErr   error
	}{
		"non exising account": {
			addr:    weavetest.NewCondition().Address(),
			wantErr: errors.ErrNotFound,
		},
		"exising account with one coin": {
			addr:      addr1,
			wantCoins: coin.Coins{&coin1},
		},
		"exising account with two coins": {
			addr: addr2,
			// Coins are stored in normalized form
			// https://github.com/iov-one/weave/pull/316#discussion_r256763396
			wantCoins: coin.Coins{&coin2_2, &coin2_1},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			coins, err := ctrl.Balance(store, tc.addr)
			if !errors.Is(tc.wantErr, err) {
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
