package cash

import (
	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
)

func init() {
	migration.MustRegister(1, &Set{}, migration.NoModification)
	migration.MustRegister(1, &Configuration{}, migration.NoModification)
}

// BucketName is where we store the balances
const BucketName = "cash"

var _ orm.CloneableData = (*Set)(nil)
var _ Coinage = (*Set)(nil)

// Validate requires that all coins are in alphabetical
func (s *Set) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", s.Metadata.Validate())
	errs = errors.AppendField(errs, "Coins", XCoins(s).Validate())
	return errs
}

// Copy makes a new set with the same coins
func (s *Set) Copy() orm.CloneableData {
	return &Set{
		Metadata: s.Metadata.Copy(),
		Coins:    XCoins(s).Clone(),
	}
}

// SetCoins allows us to modify the Set
func (s *Set) SetCoins(coins []*coin.Coin) {
	s.Coins = coins
}

// Coinage is any model that allows getting and setting coins,
// Below functions work on these models
// (oh, how I long for default implementations for interface,
// like rust traits)
type Coinage interface {
	GetCoins() []*coin.Coin
	SetCoins([]*coin.Coin)
}

// XCoins returns the stored coins cast properly
func XCoins(c Coinage) coin.Coins {
	if c == nil {
		return nil
	}
	return coin.Coins(c.GetCoins())
}

// AsCoinage will safely type-cast any value from Bucket to Coinage
func AsCoinage(obj orm.Object) Coinage {
	if obj == nil || obj.Value() == nil {
		return nil
	}
	return obj.Value().(Coinage)
}

// AsCoins will extract XCoins from any object
func AsCoins(obj orm.Object) coin.Coins {
	c := AsCoinage(obj)
	return XCoins(c)
}

// Add modifies the coinage to add Coin c
func Add(cng Coinage, c coin.Coin) error {
	cs, err := XCoins(cng).Add(c)
	if err != nil {
		return err
	}
	cng.SetCoins(cs)
	return nil
}

// Subtract modifies the coinage to remove Coin c
func Subtract(cng Coinage, c coin.Coin) error {
	return Add(cng, c.Negative())
}

// Concat combines the coins to make sure they are sorted
// and rounded off, with no duplicates or 0 values.
func Concat(cng Coinage, coins coin.Coins) error {
	joint, err := XCoins(cng).Combine(coins)
	if err != nil {
		return err
	}
	cng.SetCoins(joint)
	return nil
}

// NewWallet creates an empty wallet with this address serves as an object for
// the bucket.
// NewWallet wraps Set into an object for the Bucket.
func NewWallet(key weave.Address) orm.Object {
	return orm.NewSimpleObj(key, &Set{
		Metadata: &weave.Metadata{Schema: 1},
	})
}

// WalletWith creates an wallet with a balance
func WalletWith(key weave.Address, coins ...*coin.Coin) (orm.Object, error) {
	obj := NewWallet(key)
	err := Concat(AsCoinage(obj), coins)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// Bucket is a type-safe wrapper around orm.Bucket
type Bucket struct {
	orm.Bucket
}

var _ WalletBucket = Bucket{}

// NewBucket initializes a cash.Bucket with default name
func NewBucket() Bucket {
	return Bucket{
		Bucket: migration.NewBucket("cash", BucketName, NewWallet(nil)),
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

// WalletBucket is what we expect to be able to do with wallets
// The object it returns must support AsSet (only checked runtime :()
type WalletBucket interface {
	GetOrCreate(db weave.KVStore, key weave.Address) (orm.Object, error)
	Get(db weave.ReadOnlyKVStore, key []byte) (orm.Object, error)
	Save(db weave.KVStore, obj orm.Object) error
}

// ValidateWalletBucket makes sure that it supports AsCoinage
// objects, unfortunately this check is done runtime....
//
// panics on error (meant as a sanity check in init)
func ValidateWalletBucket(bucket WalletBucket) {
	// runtime type-check the bucket....
	db := store.MemStore()
	key := weave.NewAddress([]byte("foo"))
	obj, err := bucket.GetOrCreate(db, key)
	if err != nil {
		panic(err)
	}
	if obj == nil || obj.Value() == nil {
		panic("doensn't create anything")
	}
	// this panics if bad type
	AsCoinage(obj)
}
