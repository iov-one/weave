package client

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/cash"
)

var defaults = coin.Coin{
	Ticker:     "IOV",
	Whole:      123456789,
	Fractional: 5555555,
}

func toWeaveAddress(t *testing.T, addr string) weave.Address {
	d, err := hex.DecodeString(addr)
	assert.Nil(t, err)
	return d
}

func wsFromFile(t *testing.T, wsFile string) WalletStore {
	w := WalletStore{}
	err := w.LoadFromFile(wsFile, defaults)
	assert.Nil(t, err)
	t.Log(ToString(w))

	return w
}

func wsFromJSON(t *testing.T, ws json.RawMessage) WalletStore {
	w := WalletStore{}
	err := w.LoadFromJSON(ws, defaults)
	assert.Nil(t, err)
	t.Log(ToString(w))

	return w
}

func wsFromGenesisFile(t *testing.T, wsFile string) WalletStore {
	w := WalletStore{}
	err := w.LoadFromGenesisFile(wsFile, defaults)
	assert.Nil(t, err)
	t.Log(ToString(w))

	return w
}

func TestMergeWalletStore(t *testing.T) {
	w1 := wsFromGenesisFile(t, "./testdata/genesis.json")
	w2 := wsFromFile(t, "./testdata/wallets.json")
	expected := WalletStore{
		Wallets: []cash.GenesisAccount{
			{
				Address: toWeaveAddress(t, "3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A"),
				Set: cash.Set{
					Coins: []*coin.Coin{
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
					Coins: []*coin.Coin{
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
					Coins: []*coin.Coin{
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
					Coins: []*coin.Coin{
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
	assert.Equal(t, expected, actual)
}

func TestMergeWithEmptyWallet(t *testing.T) {
	w1 := wsFromJSON(t, []byte(`{}`))
	w2 := wsFromFile(t, "./testdata/wallets.json")

	expected := WalletStore{
		Wallets: []cash.GenesisAccount{
			{
				Address: toWeaveAddress(t, "CE5D5A5CA8C7D545D7756D3677234D81622BA297"),
				Set: cash.Set{
					Coins: []*coin.Coin{
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
					Coins: []*coin.Coin{
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
	assert.Equal(t, expected, actual)
}

func TestDefaultValues(t *testing.T) {
	actual := wsFromFile(t, "./testdata/wallets_extra.json")
	expected := WalletStore{
		Wallets: []cash.GenesisAccount{
			{
				Address: toWeaveAddress(t, "3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A"),
				Set: cash.Set{
					Coins: []*coin.Coin{
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
					Coins: []*coin.Coin{
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
					Coins: []*coin.Coin{
						{
							Ticker:     "IOV",
							Whole:      123456789,
							Fractional: 42,
						},
					},
				},
			},
			{
				Address: toWeaveAddress(t, "5AC5F736DB0E083D2316E1C5BFC141CC0C669F84"),
				Set: cash.Set{
					Coins: []*coin.Coin{
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
					Coins: []*coin.Coin{
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

	assert.Equal(t, expected, actual)
}

func TestKeyGen(t *testing.T) {
	useCases := map[string]struct {
		W string
		N int
	}{
		"empty":  {`{}`, 0},
		"single": {`{"cash":[{}]}`, 1},
	}

	for testName, useCase := range useCases {
		t.Run(testName, func(t *testing.T) {
			w := wsFromJSON(t, []byte(useCase.W))
			assert.Equal(t, useCase.N, len(w.Keys))
		})
	}
}
