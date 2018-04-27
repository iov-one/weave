package cash

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/confio/weave"
	"github.com/confio/weave/store"
	"github.com/confio/weave/x"
)

func getWallet(kv weave.KVStore, addr weave.Address) x.Coins {
	bucket := NewBucket()
	res, err := bucket.Get(kv, addr)
	if err != nil {
		panic(err) // testing only
	}
	return AsCoins(res)
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

	type issueCmd struct {
		addr   weave.Address
		amount x.Coin
		isErr  bool
	}

	type checkCmd struct {
		addr       weave.Address
		isNil      bool
		contains   []x.Coin
		notContain []x.Coin
	}

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

	kv := store.MemStore()
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

	// can't send empty
	err := controller.MoveCoins(kv, addr, addr2, send)
	require.Error(t, err)
	// so we issue money
	err = controller.IssueCoins(kv, addr, bank)
	require.NoError(t, err)

	// proper move
	err = controller.MoveCoins(kv, addr, addr2, send)
	require.NoError(t, err)
	w := getWallet(kv, addr)
	require.NotNil(t, w)
	assert.True(t, w.Contains(x.NewCoin(49700, 0, cc)))
	w2 := getWallet(kv, addr2)
	require.NotNil(t, w2)
	assert.True(t, w2.Contains(send))
	w3 := getWallet(kv, addr3)
	require.Nil(t, w3)

	// cannot send negative, zero
	err = controller.MoveCoins(kv, addr2, addr3, send.Negative())
	assert.Error(t, err)
	err = controller.MoveCoins(kv, addr2, addr3, x.NewCoin(0, 0, cc))
	assert.Error(t, err)
	w2 = getWallet(kv, addr2)
	assert.True(t, w2.Contains(send))

	// cannot send too much or no currency
	err = controller.MoveCoins(kv, addr2, addr3, bank)
	assert.Error(t, err)
	err = controller.MoveCoins(kv, addr2, addr3, x.NewCoin(5, 0, "BAD"))
	assert.Error(t, err)
	w2 = getWallet(kv, addr2)
	assert.True(t, w2.Contains(send))

	// send all coins
	err = controller.MoveCoins(kv, addr2, addr3, send)
	assert.NoError(t, err)
	w2 = getWallet(kv, addr2)
	assert.True(t, w2.IsEmpty())
	w3 = getWallet(kv, addr3)
	assert.True(t, w3.Contains(send))

	// TODO: check overflow?
}
