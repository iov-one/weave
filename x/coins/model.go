package coins

import (
	"github.com/confio/weave"
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
	set   Set
}

// GetWallet loads this Wallet if present, or returns nil if missing
func GetWallet(store weave.KVStore, key Key) *Wallet {
	bz := store.Get(key)
	if bz == nil {
		return nil
	}

	var data Set
	weave.MustUnmarshal(&data, bz)

	return &Wallet{
		store: store,
		key:   key,
		set:   data,
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
			set:   Set{},
		}
	}
	return res
}

// Save writes the current Wallet state to the backing store
// panics if invalid state
func (u *Wallet) Save() {
	// TODO: MustValidate
	err := u.set.Validate()
	if err != nil {
		panic(err)
	}

	// TODO: MustMarshal
	value, err := u.set.Marshal()
	if err != nil {
		panic(err)
	}

	u.store.Set(u.key, value)
}
