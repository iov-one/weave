package coins

import (
	"testing"

	"github.com/confio/weave"
	"github.com/confio/weave/crypto"
	"github.com/confio/weave/store"
	"github.com/confio/weave/x"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueCoins(t *testing.T) {
	kv := store.MemStore()
	addr := makeAddress()
	addr2 := makeAddress()

	plus := x.NewCoin(500, 1000, "FOO")
	minus := x.NewCoin(-400, -600, "FOO")
	total := x.NewCoin(100, 400, "FOO")
	other := x.NewCoin(1, 0, "DING")

	assert.Nil(t, GetWallet(kv, NewKey(addr)))
	assert.Nil(t, GetWallet(kv, NewKey(addr2)))

	// issue positive
	err := IssueCoins(kv, addr, plus)
	require.NoError(t, err)
	w := GetWallet(kv, NewKey(addr))
	require.NotNil(t, w)
	assert.True(t, w.Coins().Contains(plus), "%#v", w.Coins())
	assert.True(t, w.Coins().Contains(total))
	assert.False(t, w.Coins().Contains(other))
	assert.Nil(t, GetWallet(kv, NewKey(addr2)))

	// issue negative
	err = IssueCoins(kv, addr, minus)
	require.NoError(t, err)
	w = GetWallet(kv, NewKey(addr))
	require.NotNil(t, w)
	assert.False(t, w.Coins().Contains(plus))
	assert.True(t, w.Coins().Contains(total))
	assert.False(t, w.Coins().Contains(other))
	assert.Nil(t, GetWallet(kv, NewKey(addr2)))

	// issue to other wallet
	err = IssueCoins(kv, addr2, other)
	require.NoError(t, err)
	w = GetWallet(kv, NewKey(addr))
	require.NotNil(t, w)
	assert.True(t, w.Coins().Contains(total))
	assert.False(t, w.Coins().Contains(other))
	w2 := GetWallet(kv, NewKey(addr2))
	require.NotNil(t, w2)
	assert.False(t, w2.Coins().Contains(total))
	assert.True(t, w2.Coins().Contains(other))

	// set to zero is fine
	err = IssueCoins(kv, addr2, other.Negative())
	require.NoError(t, err)
	w2 = GetWallet(kv, NewKey(addr2))
	require.NotNil(t, w2)
	assert.True(t, w2.Coins().IsEmpty())

	// overflow is rejected
	err = IssueCoins(kv, addr, x.NewCoin(x.MaxInt, 0, "FOO"))
	assert.Error(t, err)
	w = GetWallet(kv, NewKey(addr))
	require.NotNil(t, w)
	assert.True(t, w.Coins().Equals(mustCombineCoins(total)))
}

func TestMoveCoins(t *testing.T) {
	kv := store.MemStore()
	addr := makeAddress()
	addr2 := makeAddress()
	addr3 := makeAddress()
	k, k2, k3 := NewKey(addr), NewKey(addr2), NewKey(addr3)

	cc := "MONY"
	bank := x.NewCoin(50000, 0, cc)
	send := x.NewCoin(300, 0, cc)

	// can't send empty
	err := MoveCoins(kv, addr, addr2, send)
	require.Error(t, err)
	// so we issue money
	err = IssueCoins(kv, addr, bank)
	require.NoError(t, err)

	// proper move
	err = MoveCoins(kv, addr, addr2, send)
	require.NoError(t, err)
	w := GetWallet(kv, k)
	require.NotNil(t, w)
	assert.True(t, w.Coins().Contains(x.NewCoin(49700, 0, cc)))
	w2 := GetWallet(kv, k2)
	require.NotNil(t, w2)
	assert.True(t, w2.Coins().Contains(send))
	w3 := GetWallet(kv, k3)
	require.Nil(t, w3)

	// cannot send negative, zero
	err = MoveCoins(kv, addr2, addr3, send.Negative())
	assert.Error(t, err)
	err = MoveCoins(kv, addr2, addr3, x.NewCoin(0, 0, cc))
	assert.Error(t, err)
	w2 = GetWallet(kv, k2)
	assert.True(t, w2.Coins().Contains(send))

	// cannot send too much or no currency
	err = MoveCoins(kv, addr2, addr3, bank)
	assert.Error(t, err)
	err = MoveCoins(kv, addr2, addr3, x.NewCoin(5, 0, "BAD"))
	assert.Error(t, err)
	w2 = GetWallet(kv, k2)
	assert.True(t, w2.Coins().Contains(send))

	// send all coins
	err = MoveCoins(kv, addr2, addr3, send)
	assert.NoError(t, err)
	w2 = GetWallet(kv, k2)
	assert.True(t, w2.Coins().IsEmpty())
	w3 = GetWallet(kv, k3)
	assert.True(t, w3.Coins().Contains(send))

	// TODO: check overflow?
}

func makeAddress() weave.Address {
	priv := crypto.GenPrivKeyEd25519()
	pub := priv.PublicKey()
	return pub.Address()
}
