package cash

import (
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

	kv := store.MemStore()
	_, perm := helpers.MakeKey()
	_, perm2 := helpers.MakeKey()
	addr := perm.Address()
	addr2 := perm2.Address()

	controller := NewController(NewBucket())

	plus := x.NewCoin(500, 1000, "FOO")
	minus := x.NewCoin(-400, -600, "FOO")
	total := x.NewCoin(100, 400, "FOO")
	other := x.NewCoin(1, 0, "DING")

	assert.Nil(t, getWallet(kv, addr))
	assert.Nil(t, getWallet(kv, addr2))

	// issue positive
	err := controller.IssueCoins(kv, addr, plus)
	require.NoError(t, err)
	w := getWallet(kv, addr)
	require.NotNil(t, w)
	assert.True(t, w.Contains(plus), "%#v", w)
	assert.True(t, w.Contains(total))
	assert.False(t, w.Contains(other))
	assert.Nil(t, getWallet(kv, addr2))

	// issue negative
	err = controller.IssueCoins(kv, addr, minus)
	require.NoError(t, err)
	w = getWallet(kv, addr)
	require.NotNil(t, w)
	assert.False(t, w.Contains(plus))
	assert.True(t, w.Contains(total))
	assert.False(t, w.Contains(other))
	assert.Nil(t, getWallet(kv, addr2))

	// issue to other wallet
	err = controller.IssueCoins(kv, addr2, other)
	require.NoError(t, err)
	w = getWallet(kv, addr)
	require.NotNil(t, w)
	assert.True(t, w.Contains(total))
	assert.False(t, w.Contains(other))
	w2 := getWallet(kv, addr2)
	require.NotNil(t, w2)
	assert.False(t, w2.Contains(total))
	assert.True(t, w2.Contains(other))

	// set to zero is fine
	err = controller.IssueCoins(kv, addr2, other.Negative())
	require.NoError(t, err)
	// object is stored in db
	res, err := controller.bucket.Get(kv, addr2)
	require.NoError(t, err)
	require.NotNil(t, res)
	// but coins are empty
	w2 = getWallet(kv, addr2)
	assert.True(t, w2.IsEmpty())

	// overflow is rejected
	err = controller.IssueCoins(kv, addr, x.NewCoin(x.MaxInt, 0, "FOO"))
	assert.Error(t, err)
	w = getWallet(kv, addr)
	require.NotNil(t, w)
	assert.True(t, w.Equals(mustCombineCoins(total)))
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
