package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/confio/weave"
	"github.com/confio/weave/x"
	"github.com/iov-one/bcp-demo/x/namecoin"
	tmtype "github.com/tendermint/tendermint/types"
)

// WalletStore represents a list of wallets from a tendermint genesis file
// It also contains private keys generated for wallets without an Address
type WalletStore struct {
	Wallets []namecoin.GenesisAccount `json:"wallets"`
	Keys    []*PrivateKey             `json:"-"`
}

// MergeWalletStore merges two WalletStore
func MergeWalletStore(w1, w2 WalletStore) WalletStore {
	combinedWallets := append(w1.Wallets, w2.Wallets...)
	combinedKeys := append(w1.Keys, w2.Keys...)
	return WalletStore{
		Wallets: combinedWallets,
		Keys:    combinedKeys,
	}
}

// LoadFromJSON loads a wallet from a json stream
// It will generate private keys for wallets without an Address
func (w *WalletStore) LoadFromJSON(msg json.RawMessage, defaults x.Coin) error {
	fmt.Printf("\nLoading new wallets from JSON %s\n", string(msg))

	if len(msg) == 0 {
		*w = WalletStore{}
		return nil
	}

	var toAdd WalletRequests
	err := json.Unmarshal(msg, &toAdd)
	if err != nil {
		return err
	}

	*w = toAdd.Normalize(defaults)
	return nil
}

// LoadFromFile loads a wallet from a file
// It will generate private keys for wallets without an Address
func (w *WalletStore) LoadFromFile(file string, defaults x.Coin) error {
	fmt.Printf("\nLoading new wallets from %s\n", file)
	newWallet, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	return w.LoadFromJSON(newWallet, defaults)
}

// LoadFromGenesisFile loads a wallet from a tendermint genesis file
// It will generate private keys for wallets without an Address
func (w *WalletStore) LoadFromGenesisFile(file string, defaults x.Coin) error {
	fmt.Printf("Loading genesis file from %s\n", file)
	genesis, err := tmtype.GenesisDocFromFile(file)
	if err != nil {
		return err
	}

	return w.LoadFromJSON(genesis.AppState(), defaults)
}

// MaybeCoin is like x.Coin, but with pointers instead
// This allows to distinguish between set values and missing ones
type MaybeCoin struct {
	Whole      *int64  `json:"whole,omitempty"`
	Fractional *int64  `json:"fractional,omitempty"`
	Ticker     *string `json:"ticker,omitempty"`
	Issuer     *string `json:"issuer,omitempty"`
}

// WithDefaults fills the gaps in a maybe coin by replacing
// missing values with default ones
func (m MaybeCoin) WithDefaults(defaults x.Coin) x.Coin {
	res := defaults
	// apply all set values, even if they are the zero value
	if m.Whole != nil {
		res.Whole = *m.Whole
	}
	if m.Fractional != nil {
		res.Fractional = *m.Fractional
	}
	if m.Ticker != nil {
		res.Ticker = *m.Ticker
	}
	if m.Issuer != nil {
		res.Issuer = *m.Issuer
	}
	return res
}

// WalletRequests contains a collection of MaybeWalletRequest
type WalletRequests struct {
	Wallets []WalletRequest `json:"wallets"`
}

// WalletRequest is like GenesisAccount, but using pointers
// To differentiate between 0 and missing
type WalletRequest struct {
	Address weave.Address `json:"address"`
	Name    string        `json:"name"`
	Coins   MaybeCoins    `json:"coins,omitempty"`
}

// WalletResponse is a response on a query for a wallet
type WalletResponse struct {
	Address weave.Address
	Wallet  namecoin.Wallet
	Height  int64
}

// Normalize Creates a WalletStore with defaulted Wallets and Generated Keys
func (w WalletRequests) Normalize(defaults x.Coin) WalletStore {
	out := WalletStore{
		Wallets: make([]namecoin.GenesisAccount, len(w.Wallets)),
	}

	for i, w := range w.Wallets {
		var newKey *PrivateKey
		out.Wallets[i], newKey = w.Normalize(defaults)

		if newKey != nil {
			out.Keys = append(out.Keys, newKey)
		}
	}

	return out
}

// Normalize returns corresponding namecoin.GenesisAccount
// with default values. It will generate private keys when there is no Address
func (w WalletRequest) Normalize(defaults x.Coin) (namecoin.GenesisAccount, *PrivateKey) {
	var coins x.Coins
	if len(w.Coins) == 0 {
		coins = x.Coins{defaults.Clone()}
	} else {
		for _, coin := range w.Coins {
			c := coin.WithDefaults(defaults)
			coins = append(coins, &c)
		}
	}

	addr := w.Address
	var privKey *PrivateKey // generated key if any
	if len(addr) == 0 {
		privKey = GenPrivateKey()
		addr = privKey.PublicKey().Address()

		fmt.Printf("Generating private key: %X\n\n", privKey)
	}

	return namecoin.GenesisAccount{
		Address: addr,
		Wallet: &namecoin.Wallet{
			Name:  w.Name,
			Coins: coins,
		},
	}, privKey
}

type MaybeCoins []*MaybeCoin

// WithCoinDefaults applies WithDefault to a collection of coins
// If no defaults are found for a given coin, it will be ignored
func WithCoinDefaults(manyCoins MaybeCoins, defaults x.Coins) x.Coins {
	var coins x.Coins
	if len(manyCoins) == 0 {
		coins = defaults.Clone()
	} else {
		for _, coin := range manyCoins {
			for _, defaultCoin := range defaults {
				if defaultCoin.Ticker == *coin.Ticker {
					coinWithDefault := coin.WithDefaults(*defaultCoin)
					coins = append(coins, &coinWithDefault)
					break
				}
			}
		}
	}

	return coins
}
