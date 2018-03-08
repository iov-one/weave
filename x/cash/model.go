package cash

import (
	"errors"

	"github.com/confio/weave"
	"github.com/confio/weave/orm"
	"github.com/confio/weave/x"
)

// BucketName is where we store the balances
const BucketName = "cash"

//---- Set

// Validate requires that all coins are in alphabetical
func (s *Set) Validate() error {
	return x.Coins(s.GetCoins()).Validate()
}

// Copy makes a new set with the same coins
func (s *Set) Copy() *Set {
	return &Set{
		Coins: x.Coins(s.GetCoins()).Clone(),
	}
}

//--- Wallet (Set object, wallet + key)

// Wallet is the actual object that we want to pass around
// in our code. It contains a set of coins, as well as the
// address. It is connected to the Bucket to easily manipulate
// state.
//
// Wallet is a type-safe wrapper around orm.SimpleObj
type Wallet struct {
	key   []byte
	value *Set
}

// NewWallet creates an empty wallet with this address
func NewWallet(key weave.Address, coins ...*x.Coin) *Wallet {
	res := &Wallet{key, new(Set)}
	if coins != nil {
		err := res.Concat(coins)
		if err != nil {
			panic(err)
		}
	}
	return res
}

// Value gets the value stored in the object
func (w Wallet) Value() weave.Persistent {
	return w.value
}

// Key returns the key to store the object under
func (w Wallet) Key() []byte {
	return w.key
}

// Validate makes sure the fields aren't empty.
// And delegates to the value validator if present
func (w Wallet) Validate() error {
	if len(w.key) == 0 {
		return errors.New("Missing key")
	}
	return w.value.Validate()
}

// SetKey may be used to update a simple obj key
func (w *Wallet) SetKey(key []byte) {
	w.key = key
}

// Clone will make a copy of this object
func (w *Wallet) Clone() orm.Object {
	res := &Wallet{
		value: w.value.Copy(),
	}
	// only copy key if non-nil
	if len(w.key) > 0 {
		res.key = append([]byte(nil), w.key...)
	}
	return res
}

// Coins returns the coins stored in the wallet
func (w Wallet) Coins() x.Coins {
	return x.Coins(w.value.GetCoins())
}

// Add modifies the wallet to add Coin c
func (w *Wallet) Add(c x.Coin) error {
	cs, err := w.Coins().Add(c)
	if err != nil {
		return err
	}
	w.value.Coins = cs
	return nil
}

// Subtract modifies the wallet to remove Coin c
func (w *Wallet) Subtract(c x.Coin) error {
	return w.Add(c.Negative())
}

// Concat combines the coins to make sure they are sorted
// and rounded off, with no duplicates or 0 values.
//
// TODO: can we make this simpler??? join with copy
func (w *Wallet) Concat(coins x.Coins) error {
	joint, err := w.Coins().Combine(coins)
	if err != nil {
		return err
	}
	w.value.Coins = joint
	return nil
}

//--- cash.Bucket - type-safe bucket

// Bucket is a type-safe wrapper around orm.Bucket
type Bucket struct {
	orm.Bucket
}

// NewBucket initializes a cash.Bucket with default name
func NewBucket() Bucket {
	return Bucket{
		Bucket: orm.NewBucket(BucketName, NewWallet(nil)),
	}
}

func (b Bucket) Get(db weave.KVStore, key weave.Address) (*Wallet, error) {
	obj, err := b.Bucket.Get(db, key)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, nil
	}
	return obj.(*Wallet), nil
}

func (b Bucket) Save(db weave.KVStore, value *Wallet) error {
	return b.Bucket.Save(db, value)
}

func (b Bucket) GetOrCreate(db weave.KVStore, key weave.Address) (*Wallet, error) {
	wallet, err := b.Get(db, key)
	if err != nil {
		return nil, err
	}
	if wallet == nil {
		wallet = NewWallet(key)
	}
	return wallet, nil
}
