package utils

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iov-one/weave"

	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/namecoin"
	"github.com/stretchr/testify/assert"
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
	w1 := wsFromGenesisFile(t, "./../testnet/testdata/genesis.json")
	w2 := wsFromFile(t, "./../testnet/testdata/wallets.json")
	expected := WalletStore{
		Wallets: []namecoin.GenesisAccount{
			namecoin.GenesisAccount{
				Address: toWeaveAddress(t, "3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A"),
				Wallet: &namecoin.Wallet{
					Name: "admin",
					Coins: []*x.Coin{
						&x.Coin{
							Ticker:     "CASH",
							Whole:      123456789,
							Fractional: 5555555,
						},
					},
				},
			},
			namecoin.GenesisAccount{
				Address: toWeaveAddress(t, "12AFFBF6012FD2DF21416582DC80CBF1EFDF2460"),
				Wallet: &namecoin.Wallet{
					Name: "second",
					Coins: []*x.Coin{
						&x.Coin{
							Ticker:     "CASH",
							Whole:      987654321,
							Fractional: 5555555,
						},
					},
				},
			},
			namecoin.GenesisAccount{
				Address: toWeaveAddress(t, "CE5D5A5CA8C7D545D7756D3677234D81622BA297"),
				Wallet: &namecoin.Wallet{
					Name: "alice",
					Coins: []*x.Coin{
						&x.Coin{
							Ticker:     "IOV",
							Whole:      123456789,
							Fractional: 5555555,
						},
					},
				},
			},
			namecoin.GenesisAccount{
				Address: toWeaveAddress(t, "D4821FD051696273D09E1FBAD0EBE5B5060787A7"),
				Wallet: &namecoin.Wallet{
					Name: "bert",
					Coins: []*x.Coin{
						&x.Coin{
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
	w2 := wsFromFile(t, "./../testnet/testdata/wallets.json")

	expected := WalletStore{
		Wallets: []namecoin.GenesisAccount{
			namecoin.GenesisAccount{
				Address: toWeaveAddress(t, "CE5D5A5CA8C7D545D7756D3677234D81622BA297"),
				Wallet: &namecoin.Wallet{
					Name: "alice",
					Coins: []*x.Coin{
						&x.Coin{
							Ticker:     "IOV",
							Whole:      123456789,
							Fractional: 5555555,
						},
					},
				},
			},
			namecoin.GenesisAccount{
				Address: toWeaveAddress(t, "D4821FD051696273D09E1FBAD0EBE5B5060787A7"),
				Wallet: &namecoin.Wallet{
					Name: "bert",
					Coins: []*x.Coin{
						&x.Coin{
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
	actual := wsFromFile(t, "./../testnet/testdata/wallets_extra.json")
	expected := WalletStore{
		Wallets: []namecoin.GenesisAccount{
			namecoin.GenesisAccount{
				Address: toWeaveAddress(t, "3AFCDAB4CFBF066E959D139251C8F0EE91E99D5A"),
				Wallet: &namecoin.Wallet{
					Name: "first",
					Coins: []*x.Coin{
						&x.Coin{
							Ticker:     "IOV",
							Whole:      123456789,
							Fractional: 5555555,
						},
					},
				},
			},
			namecoin.GenesisAccount{
				Address: toWeaveAddress(t, "CE5D5A5CA8C7D545D7756D3677234D81622BA297"),
				Wallet: &namecoin.Wallet{
					Name: "second",
					Coins: []*x.Coin{
						&x.Coin{
							Ticker:     "IOV",
							Whole:      17,
							Fractional: 5555555,
						},
					},
				},
			},
			namecoin.GenesisAccount{
				Address: toWeaveAddress(t, "D4821FD051696273D09E1FBAD0EBE5B5060787A7"),
				Wallet: &namecoin.Wallet{
					Name: "third",
					Coins: []*x.Coin{
						&x.Coin{
							Ticker:     "IOV",
							Whole:      123456789,
							Fractional: 42,
							Issuer:     "IOV-ONE",
						},
					},
				},
			},
			namecoin.GenesisAccount{
				Address: toWeaveAddress(t, "5AC5F736DB0E083D2316E1C5BFC141CC0C669F84"),
				Wallet: &namecoin.Wallet{
					Name: "fourth",
					Coins: []*x.Coin{
						&x.Coin{
							Ticker:     "IOV",
							Whole:      0,
							Fractional: 0,
						},
					},
				},
			},
			namecoin.GenesisAccount{
				Address: toWeaveAddress(t, "12AFFBF6012FD2DF21416582DC80CBF1EFDF2460"),
				Wallet: &namecoin.Wallet{
					Name: "fifth",
					Coins: []*x.Coin{
						&x.Coin{
							Ticker:     "ETH",
							Whole:      123456789,
							Fractional: 5555555,
						},
						&x.Coin{
							Ticker:     "CASH",
							Whole:      123456789,
							Fractional: 5555555,
						},
						&x.Coin{
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
		{`{"wallets":[{"name": "alice"}]}`, 1},
		{`{"wallets":[{"name": "alice"},{"name": "dora"}]}`, 2},
		{`{"wallets":[{"name": "alice"},{"name": "dora"},{"name": "bert"}]}`, 3},
		{`{"wallets":[{"name": "alice"},{"name": "dora"},{"name": "bert"},{"name": "charlie"}]}`, 4},
	}

	for _, useCase := range useCases {
		w := wsFromJSON(t, []byte(useCase.W))
		assert.Len(t, w.Keys, useCase.N)
	}
}
