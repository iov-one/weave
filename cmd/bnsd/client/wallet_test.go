package client

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/nft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var defaults = x.Coin{
	Ticker:     "IOV",
	Whole:      123456789,
	Fractional: 5555555,
}

func toWeaveAddress(t *testing.T, addr string) weave.Address {
	d, err := hex.DecodeString(addr)
	assert.Nil(t, err, "failed to decode weave address from string")
	return d
}

func wsFromFile(t *testing.T, wsFile string) WalletStore {
	w := WalletStore{}
	err := w.LoadFromFile(wsFile, defaults)
	require.Nil(t, err, fmt.Sprintf("loading new wallet store from %s\n", wsFile))
	t.Log(ToString(w))

	return w
}

func wsFromJSON(t *testing.T, ws json.RawMessage) WalletStore {
	w := WalletStore{}
	err := w.LoadFromJSON(ws, defaults)
	require.Nil(t, err, fmt.Sprintf("loading new wallet store from JSON: %s\n", string(ws)))
	t.Log(ToString(w))

	return w
}

func wsFromGenesisFile(t *testing.T, wsFile string) WalletStore {
	w := WalletStore{}
	err := w.LoadFromGenesisFile(wsFile, defaults)
	require.Nil(t, err, fmt.Sprintf("loading new wallet store from %s\n", wsFile))
	t.Log(ToString(w))

	return w
}

func TestMergeWalletStore(t *testing.T) {
	nft.RegisterAction(nft.DefaultActions...)
	w1 := wsFromGenesisFile(t, "./testdata/genesis.json")
	w2 := wsFromFile(t, "./testdata/wallets.json")
	expected := WalletStore{
		Wallets: []cash.GenesisAccount{
			{
				Address: toWeaveAddress(t, "3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A"),
				Set: cash.Set{
					Coins: []*x.Coin{
						{
							Ticker:     "CASH",
							Whole:      123456789,
							Fractional: 5555555,
						},
					},
				},
			},
			{
				Address: toWeaveAddress(t, "12AFFBF6012FD2DF21416582DC80CBF1EFDF2460"),
				Set: cash.Set{
					Coins: []*x.Coin{
						{
							Ticker:     "CASH",
							Whole:      987654321,
							Fractional: 5555555,
						},
					},
				},
			},
			{
				Address: toWeaveAddress(t, "CE5D5A5CA8C7D545D7756D3677234D81622BA297"),
				Set: cash.Set{
					Coins: []*x.Coin{
						{
							Ticker:     "IOV",
							Whole:      123456789,
							Fractional: 5555555,
						},
					},
				},
			},
			{
				Address: toWeaveAddress(t, "D4821FD051696273D09E1FBAD0EBE5B5060787A7"),
				Set: cash.Set{
					Coins: []*x.Coin{
						{
							Ticker:     "IOV",
							Whole:      123456789,
							Fractional: 5555555,
						},
					},
				},
			},
		},
	}

	actual := MergeWalletStore(w1, w2)
	assert.EqualValues(t, expected, actual, ToString(expected), ToString(actual))
}

func TestMergeWithEmptyWallet(t *testing.T) {
	w1 := wsFromJSON(t, []byte(`{}`))
	w2 := wsFromFile(t, "./testdata/wallets.json")

	expected := WalletStore{
		Wallets: []cash.GenesisAccount{
			{
				Address: toWeaveAddress(t, "CE5D5A5CA8C7D545D7756D3677234D81622BA297"),
				Set: cash.Set{
					Coins: []*x.Coin{
						{
							Ticker:     "IOV",
							Whole:      123456789,
							Fractional: 5555555,
						},
					},
				},
			},
			{
				Address: toWeaveAddress(t, "D4821FD051696273D09E1FBAD0EBE5B5060787A7"),
				Set: cash.Set{
					Coins: []*x.Coin{
						{
							Ticker:     "IOV",
							Whole:      123456789,
							Fractional: 5555555,
						},
					},
				},
			},
		},
	}

	actual := MergeWalletStore(w1, w2)
	assert.EqualValues(t, expected, actual, ToString(expected), ToString(actual))
}

func TestDefaultValues(t *testing.T) {
	actual := wsFromFile(t, "./testdata/wallets_extra.json")
	expected := WalletStore{
		Wallets: []cash.GenesisAccount{
			{
				Address: toWeaveAddress(t, "3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A"),
				Set: cash.Set{
					Coins: []*x.Coin{
						{
							Ticker:     "IOV",
							Whole:      123456789,
							Fractional: 5555555,
						},
					},
				},
			},
			{
				Address: toWeaveAddress(t, "CE5D5A5CA8C7D545D7756D3677234D81622BA297"),
				Set: cash.Set{
					Coins: []*x.Coin{
						{
							Ticker:     "IOV",
							Whole:      17,
							Fractional: 5555555,
						},
					},
				},
			},
			{
				Address: toWeaveAddress(t, "D4821FD051696273D09E1FBAD0EBE5B5060787A7"),
				Set: cash.Set{
					Coins: []*x.Coin{
						{
							Ticker:     "IOV",
							Whole:      123456789,
							Fractional: 42,
							Issuer:     "IOV-ONE",
						},
					},
				},
			},
			{
				Address: toWeaveAddress(t, "5AC5F736DB0E083D2316E1C5BFC141CC0C669F84"),
				Set: cash.Set{
					Coins: []*x.Coin{
						{
							Ticker:     "IOV",
							Whole:      0,
							Fractional: 0,
						},
					},
				},
			},
			{
				Address: toWeaveAddress(t, "12AFFBF6012FD2DF21416582DC80CBF1EFDF2460"),
				Set: cash.Set{
					Coins: []*x.Coin{
						{
							Ticker:     "ETH",
							Whole:      123456789,
							Fractional: 5555555,
						},
						{
							Ticker:     "CASH",
							Whole:      123456789,
							Fractional: 5555555,
						},
						{
							Ticker:     "IOV",
							Whole:      1000,
							Fractional: 5555555,
						},
					},
				},
			},
		},
	}

	assert.EqualValues(t, expected, actual, ToString(expected), ToString(actual))
}

func TestKeyGen(t *testing.T) {
	useCases := []struct {
		W string
		N int
	}{
		{`{}`, 0},
		{`{"cash":[{}]}`, 1},
		//{`{"cash":[{},{}]}`, 2},
		//{`{"cash":[{"name": "alice"},{"name": "dora"},{"name": "bert"}]}`, 3},
		//{`{"cash":[{"name": "alice"},{"name": "dora"},{"name": "bert"},{"name": "charlie"}]}`, 4},
	}

	for _, useCase := range useCases {
		w := wsFromJSON(t, []byte(useCase.W))
		assert.Len(t, w.Keys, useCase.N)
	}
}
