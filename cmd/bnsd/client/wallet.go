package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/x/cash"
	tmtype "github.com/tendermint/tendermint/types"
)

// WalletStore represents a list of wallets from a tendermint genesis file
// It also contains private keys generated for wallets without an Address
type WalletStore struct {
	Wallets []cash.GenesisAccount `json:"wallets"`
	Keys    []*PrivateKey         `json:"-"`
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
func (w *WalletStore) LoadFromJSON(msg json.RawMessage, defaults coin.Coin) error {
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
func (w *WalletStore) LoadFromFile(file string, defaults coin.Coin) error {
	fmt.Printf("\nLoading new wallets from %s\n", file)
	newWallet, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	return w.LoadFromJSON(newWallet, defaults)
}

// LoadFromGenesisFile loads a wallet from a tendermint genesis file
// It will generate private keys for wallets without an Address
func (w *WalletStore) LoadFromGenesisFile(file string, defaults coin.Coin) error {
	fmt.Printf("Loading genesis file from %s\n", file)
	genesis, err := tmtype.GenesisDocFromFile(file)
	if err != nil {
		return err
	}

	return w.LoadFromJSON(genesis.AppState, defaults)
}

// MaybeCoin is like coin.Coin, but with pointers instead
// This allows to distinguish between set values and missing ones
type MaybeCoin struct {
	Whole      *int64  `json:"whole,omitempty"`
	Fractional *int64  `json:"fractional,omitempty"`
	Ticker     *string `json:"ticker,omitempty"`
}

// WithDefaults fills the gaps in a maybe coin by replacing
// missing values with default ones
func (m MaybeCoin) WithDefaults(defaults coin.Coin) coin.Coin {
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
	return res
}

// WalletRequests contains a collection of MaybeWalletRequest
type WalletRequests struct {
	Wallets []WalletRequest `json:"cash"`
}

// WalletRequest is like GenesisAccount, but using pointers
// To differentiate between 0 and missing
type WalletRequest struct {
	Address weave.Address `json:"address"`
	Coins   MaybeCoins    `json:"coins,omitempty"`
}

// WalletResponse is a response on a query for a wallet
type WalletResponse struct {
	Address weave.Address
	Wallet  cash.Set
	Height  int64
}

// Normalize Creates a WalletStore with defaulted Wallets and Generated Keys
func (w WalletRequests) Normalize(defaults coin.Coin) WalletStore {
	out := WalletStore{
		Wallets: make([]cash.GenesisAccount, len(w.Wallets)),
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

// Normalize returns corresponding cash.GenesisAccount
// with default values. It will generate private keys when there is no Address
func (w WalletRequest) Normalize(defaults coin.Coin) (cash.GenesisAccount, *PrivateKey) {
	var coins coin.Coins
	if len(w.Coins) == 0 {
		coins = coin.Coins{defaults.Clone()}
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

	return cash.GenesisAccount{
		Address: addr,
		Set:     cash.Set{Coins: coins},
	}, privKey
}

type MaybeCoins []*MaybeCoin

func FindCoinByTicker(coins coin.Coins, ticker string) (*coin.Coin, bool) {
	for _, coin := range coins {
		if strings.EqualFold(ticker, coin.Ticker) {
			return coin, true
		}
	}
	return nil, false
}
