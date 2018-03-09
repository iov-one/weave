package cash

import (
	"github.com/confio/weave"
	"github.com/confio/weave/orm"
	"github.com/confio/weave/x"
)

// BucketName is where we store the balances
const BucketName = "cash"

//---- Set

func (s *Set) xcoins() x.Coins {
	return x.Coins(s.GetCoins())
}

// Validate requires that all coins are in alphabetical
func (s *Set) Validate() error {
	return s.xcoins().Validate()
}

// Copy makes a new set with the same coins
func (s *Set) Copy() orm.CloneableData {
	return &Set{
		Coins: s.xcoins().Clone(),
	}
}

// AsSet will safely type-cast any value from Bucket to a Set
func AsSet(obj orm.Object) *Set {
	if obj == nil || obj.Value() == nil {
		return nil
	}
	return obj.Value().(*Set)
}

//------ expose x.Coins methods

// Contains returns true if there is at least that much
// coin in the Set
func (s Set) Contains(c x.Coin) bool {
	return s.xcoins().Contains(c)
}

// IsEmpty checks if no coins in the set
func (s Set) IsEmpty() bool {
	return s.xcoins().IsEmpty()
}

// Equals checks if the coins are the same
func (s Set) Equals(coins x.Coins) bool {
	return s.xcoins().Equals(coins)
}

// Add modifies the wallet to add Coin c
func (s *Set) Add(c x.Coin) error {
	cs, err := s.xcoins().Add(c)
	if err != nil {
		return err
	}
	s.Coins = cs
	return nil
}

// Subtract modifies the wallet to remove Coin c
func (s *Set) Subtract(c x.Coin) error {
	return s.Add(c.Negative())
}

// Concat combines the coins to make sure they are sorted
// and rounded off, with no duplicates or 0 values.
func (s *Set) Concat(coins x.Coins) error {
	joint, err := s.xcoins().Combine(coins)
	if err != nil {
		return err
	}
	s.Coins = joint
	return nil
}

// NewWallet creates an empty wallet with this address
// serves as an object for the bucket
func NewWallet(key weave.Address) orm.Object {
	return orm.NewSimpleObj(key, new(Set))
}

// WalletWith creates an wallet with a balance
func WalletWith(key weave.Address, coins ...*x.Coin) (orm.Object, error) {
	obj := NewWallet(key)
	err := AsSet(obj).Concat(coins)
	if err != nil {
		return nil, err
	}
	return obj, nil
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

// GetOrCreate will return the object if found, or create one
// if not.
func (b Bucket) GetOrCreate(db weave.KVStore, key weave.Address) (orm.Object, error) {
	obj, err := b.Get(db, key)
	if err == nil && obj == nil {
		obj = NewWallet(key)
	}
	return obj, err
}
