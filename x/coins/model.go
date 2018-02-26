package coins

import (
	"github.com/confio/weave"
	"github.com/confio/weave/x"
)

//---- Key

// Key is the primary key we use to distinguish users
// This should be []byte, in order to index with our KVStore.
// Any structure to these bytes should be defined by the constructor.
//
// Question: allow objects with a Marshal method???
type Key []byte

var walletPrefix = []byte("wallet:")

// NewKey constructs the coin key from a key hash,
// by appending a prefix.
func NewKey(addr weave.Address) Key {
	bz := append(walletPrefix, addr...)
	return Key(bz)
}

//------------------ High-Level ------------------------

// Wallet is the actual object that we want to pass around
// in our code. It contains a set of coins, as well as
// logic to handle loading and saving the data to/from the
// persistent store, and helpers to manipulate state.
type Wallet struct {
	store weave.KVStore
	key   Key
	Set   Set
}

// GetWallet loads this Wallet if present, or returns nil if missing
func GetWallet(store weave.KVStore, key Key) *Wallet {
	bz := store.Get(key)
	if bz == nil {
		return nil
	}

	var data Set
	x.MustUnmarshal(&data, bz)

	return &Wallet{
		store: store,
		key:   key,
		Set:   data,
	}
}

// GetOrCreateWallet loads this Wallet if present,
// or initializes a new Wallet with this key if not present.
func GetOrCreateWallet(store weave.KVStore, key Key) *Wallet {
	res := GetWallet(store, key)
	if res == nil {
		res = &Wallet{
			store: store,
			key:   key,
			Set:   Set{},
		}
	}
	return res
}

// Save writes the current Wallet state to the backing store
// panics if invalid state
func (u *Wallet) Save() {
	value := x.MustMarshalValid(&u.Set)
	u.store.Set(u.key, value)
}

// Coins returns the coins stored in the wallet
func (u Wallet) Coins() x.Coins {
	return x.Coins(u.Set.Coins)
}

// Add modifies the wallet to add Coin c
func (u *Wallet) Add(c x.Coin) error {
	cs, err := u.Coins().Add(c)
	if err != nil {
		return err
	}
	u.Set.Coins = cs
	return nil
}

// Subtract modifies the wallet to remove Coin c
func (u *Wallet) Subtract(c x.Coin) error {
	return u.Add(c.Negative())
}

// Validate requires that all coins are in alphabetical
func (s Set) Validate() error {
	return x.Coins(s.Coins).Validate()
}

// Normalize combines the coins to make sure they are sorted
// and rounded off, with no duplicates or 0 values.
func (s Set) Normalize() (Set, error) {
	ins := make([]x.Coin, len(s.GetCoins()))
	for i, c := range s.GetCoins() {
		ins[i] = *c
	}
	coins, err := x.CombineCoins(ins...)
	if err != nil {
		return Set{}, err
	}
	return Set{Coins: coins}, nil
}
