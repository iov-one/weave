package coins

import (
	"testing"

	"github.com/confio/weave"
	"github.com/confio/weave/crypto"
	"github.com/confio/weave/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueCoins(t *testing.T) {
	kv := store.MemStore()
	addr := makeAddress()
	addr2 := makeAddress()

	plus := NewCoin(500, 1000, "FOO")
	minus := NewCoin(-400, -600, "FOO")
	total := NewCoin(100, 400, "FOO")
	other := NewCoin(1, 0, "DING")

	assert.Nil(t, GetWallet(kv, NewKey(addr)))
	assert.Nil(t, GetWallet(kv, NewKey(addr2)))

	// issue positive
	err := IssueCoins(kv, addr, plus)
	require.NoError(t, err)
	w := GetWallet(kv, NewKey(addr))
	require.NotNil(t, w)
	assert.True(t, w.Contains(plus))
	assert.True(t, w.Contains(total))
	assert.False(t, w.Contains(other))
	assert.Nil(t, GetWallet(kv, NewKey(addr2)))

	// issue negative
	err = IssueCoins(kv, addr, minus)
	require.NoError(t, err)
	w = GetWallet(kv, NewKey(addr))
	require.NotNil(t, w)
	assert.False(t, w.Contains(plus))
	assert.True(t, w.Contains(total))
	assert.False(t, w.Contains(other))
	assert.Nil(t, GetWallet(kv, NewKey(addr2)))

	// issue to other wallet
	err = IssueCoins(kv, addr2, other)
	require.NoError(t, err)
	w = GetWallet(kv, NewKey(addr))
	require.NotNil(t, w)
	assert.True(t, w.Contains(total))
	assert.False(t, w.Contains(other))
	w2 := GetWallet(kv, NewKey(addr2))
	require.NotNil(t, w2)
	assert.False(t, w2.Contains(total))
	assert.True(t, w2.Contains(other))

	// set to zero is fine
	err = IssueCoins(kv, addr2, other.Negative())
	require.NoError(t, err)
	w2 = GetWallet(kv, NewKey(addr2))
	require.NotNil(t, w2)
	assert.True(t, w2.IsEmpty())

	// overflow is rejected
	err = IssueCoins(kv, addr, NewCoin(maxInt, 0, "FOO"))
	assert.Error(t, err)
	w = GetWallet(kv, NewKey(addr))
	require.NotNil(t, w)
	assert.True(t, w.Equals(mustNewSet(total)))
}

func TestMoveCoins(t *testing.T) {
	kv := store.MemStore()
	addr := makeAddress()
	addr2 := makeAddress()
	addr3 := makeAddress()
	k, k2, k3 := NewKey(addr), NewKey(addr2), NewKey(addr3)

	cc := "MONY"
	bank := NewCoin(50000, 0, cc)
	send := NewCoin(300, 0, cc)

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
	assert.True(t, w.Contains(NewCoin(49700, 0, cc)))
	w2 := GetWallet(kv, k2)
	require.NotNil(t, w2)
	assert.True(t, w2.Contains(send))
	w3 := GetWallet(kv, k3)
	require.Nil(t, w3)

	// cannot send negative, zero
	err = MoveCoins(kv, addr2, addr3, send.Negative())
	assert.Error(t, err)
	err = MoveCoins(kv, addr2, addr3, NewCoin(0, 0, cc))
	assert.Error(t, err)
	w2 = GetWallet(kv, k2)
	assert.True(t, w2.Contains(send))

	// cannot send too much or no currency
	err = MoveCoins(kv, addr2, addr3, bank)
	assert.Error(t, err)
	err = MoveCoins(kv, addr2, addr3, NewCoin(5, 0, "BAD"))
	assert.Error(t, err)
	w2 = GetWallet(kv, k2)
	assert.True(t, w2.Contains(send))

	// send all coins
	err = MoveCoins(kv, addr2, addr3, send)
	assert.NoError(t, err)
	w2 = GetWallet(kv, k2)
	assert.True(t, w2.IsEmpty())
	w3 = GetWallet(kv, k3)
	assert.True(t, w3.Contains(send))

	// TODO: check overflow?
}

func makeAddress() weave.Address {
	priv := crypto.GenPrivKeyEd25519()
	pub := priv.PublicKey()
	return pub.Address()
}
