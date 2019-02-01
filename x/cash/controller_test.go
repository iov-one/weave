package cash

import (
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getWallet(kv weave.KVStore, addr weave.Address) x.Coins {
	bucket := NewBucket()
	res, err := bucket.Get(kv, addr)
	if err != nil {
		panic(err) // testing only
	}
	return AsCoins(res)
}

type issueCmd struct {
	addr   weave.Address
	amount x.Coin
	isErr  bool
}

type moveCmd struct {
	sender weave.Address
	rcpt   weave.Address
	amount x.Coin
	isErr  bool
}

type checkCmd struct {
	addr       weave.Address
	isNil      bool
	contains   []x.Coin
	notContain []x.Coin
}

func TestIssueCoins(t *testing.T) {
	var helpers x.TestHelpers

	_, perm := helpers.MakeKey()
	_, perm2 := helpers.MakeKey()
	addr := perm.Address()
	addr2 := perm2.Address()

	controller := NewController(NewBucket())

	plus := x.NewCoin(500, 1000, "FOO")
	minus := x.NewCoin(-400, -600, "FOO")
	total := x.NewCoin(100, 400, "FOO")
	other := x.NewCoin(1, 0, "DING")

	cases := []struct {
		issue []issueCmd
		check []checkCmd
	}{
		// issue positive
		{
			issue: []issueCmd{{addr, plus, false}},
			check: []checkCmd{
				{addr, false, []x.Coin{plus, total}, []x.Coin{other}},
				{addr2, true, nil, nil},
			},
		},
		// second issue negative
		{
			issue: []issueCmd{{addr, plus, false}, {addr, minus, false}},
			check: []checkCmd{
				{addr, false, []x.Coin{total}, []x.Coin{plus, other}},
				{addr2, true, nil, nil},
			},
		},
		// issue to two chains
		{
			issue: []issueCmd{{addr, total, false}, {addr2, other, false}},
			check: []checkCmd{
				{addr, false, []x.Coin{total}, []x.Coin{plus, other}},
				{addr2, false, []x.Coin{other}, []x.Coin{plus, total}},
			},
		},
		// set back to zero
		{
			issue: []issueCmd{{addr2, other, false}, {addr2, other.Negative(), false}},
			check: []checkCmd{
				{addr, true, nil, nil},
				{addr2, true, nil, nil},
			},
		},
		// set back to zero
		{
			issue: []issueCmd{
				{addr, total, false},
				{addr, x.NewCoin(x.MaxInt, 0, "FOO"), true}},
			check: []checkCmd{
				{addr, false, []x.Coin{total}, []x.Coin{plus, other}},
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
	var helpers x.TestHelpers

	_, perm := helpers.MakeKey()
	_, perm2 := helpers.MakeKey()
	_, perm3 := helpers.MakeKey()
	addr := perm.Address()
	addr2 := perm2.Address()
	addr3 := perm3.Address()

	controller := NewController(NewBucket())

	cc := "MONY"
	bank := x.NewCoin(50000, 0, cc)
	send := x.NewCoin(300, 0, cc)
	rem := x.NewCoin(49700, 0, cc)

	cases := []struct {
		issue issueCmd
		move  moveCmd
		check []checkCmd
	}{
		// cannot move money that you don't have
		{
			issue: issueCmd{addr3, bank, false},
			move:  moveCmd{addr, addr2, send, true},
			check: []checkCmd{
				{addr2, true, nil, nil},
				{addr3, false, []x.Coin{bank}, nil},
			},
		},
		// simple send
		{
			issue: issueCmd{addr, bank, false},
			move:  moveCmd{addr, addr2, send, false},
			check: []checkCmd{
				{addr, false, []x.Coin{rem}, []x.Coin{bank}},
				{addr2, false, []x.Coin{send}, []x.Coin{bank}},
			},
		},
		// cannot send negative
		{
			issue: issueCmd{addr, bank, false},
			move:  moveCmd{addr, addr2, send.Negative(), true},
			check: nil,
		},
		// cannot send more than you have
		{
			issue: issueCmd{addr, rem, false},
			move:  moveCmd{addr, addr2, bank, true},
			check: nil,
		},
		// cannot send zero
		{
			issue: issueCmd{addr, bank, false},
			move:  moveCmd{addr, addr2, x.NewCoin(0, 0, cc), true},
			check: nil,
		},
		// cannot send wrong currency
		{
			issue: issueCmd{addr, bank, false},
			move:  moveCmd{addr, addr2, x.NewCoin(500, 0, "BAD"), true},
			check: nil,
		},
		// send everything
		{
			issue: issueCmd{addr, bank, false},
			move:  moveCmd{addr, addr2, bank, false},
			check: []checkCmd{
				{addr, true, nil, nil},
				{addr2, false, []x.Coin{bank}, nil},
			},
		},
		// send to self
		{
			issue: issueCmd{addr, rem, false},
			move:  moveCmd{addr, addr, send, false},
			check: []checkCmd{
				{addr, false, []x.Coin{send, rem}, []x.Coin{bank}},
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
