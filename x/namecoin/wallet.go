package namecoin

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
)

const (
	// BucketNameWallet is where we store the balances
	BucketNameWallet = "wllt"
	// IndexName is the index to query wallet by name
	IndexName = "name"
)

//--- Wallet

var _ orm.CloneableData = (*Wallet)(nil)
var _ cash.Coinage = (*Wallet)(nil)
var _ Named = (*Wallet)(nil)

// SetCoins lets us modify the wallet
// and satisfy Coinage to be compatible with x/cash
func (w *Wallet) SetCoins(coins []*x.Coin) {
	w.Coins = coins
}

// Validate requires that all coins are in alphabetical
func (w *Wallet) Validate() error {
	name := w.GetName()
	if name != "" && !IsWalletName(name) {
		return ErrInvalidWalletName(name)
	}
	return cash.XCoins(w).Validate()
}

// Copy makes a new set with the same coins
func (w *Wallet) Copy() orm.CloneableData {
	return &Wallet{
		Name:  w.Name,
		Coins: cash.XCoins(w).Clone(),
	}
}

// SetName verifies the name is valid and sets it on the wallet
func (w *Wallet) SetName(name string) error {
	if w.Name != "" {
		return ErrChangeWalletName()
	}
	if !IsWalletName(name) {
		return ErrInvalidWalletName(name)
	}
	w.Name = name
	return nil
}

// AsWallet safely extracts a Wallet value from the object
func AsWallet(obj orm.Object) *Wallet {
	if obj == nil || obj.Value() == nil {
		return nil
	}
	return obj.Value().(*Wallet)
}

// AsNamed returns an object that has can get/set names
func AsNamed(obj orm.Object) Named {
	if obj == nil || obj.Value() == nil {
		return nil
	}
	return obj.Value().(Named)
}

// NewWallet creates an empty wallet with this address
// serves as an object for the bucket
func NewWallet(key weave.Address) orm.Object {
	return orm.NewSimpleObj(key, new(Wallet))
}

// WalletWith creates an wallet with a balance
func WalletWith(key weave.Address, name string, coins ...*x.Coin) (orm.Object, error) {
	obj := NewWallet(key)
	err := cash.Concat(cash.AsCoinage(obj), coins)
	if err != nil {
		return nil, err
	}
	if name != "" {
		err := AsNamed(obj).SetName(name)
		if err != nil {
			return nil, err
		}
	}
	return obj, nil
}

//--- WalletBucket - handles tokens

// WalletBucket is a type-safe wrapper around orm.Bucket
type WalletBucket struct {
	orm.Bucket
}

var _ cash.WalletBucket = WalletBucket{}
var _ NamedBucket = WalletBucket{}

// NewWalletBucket initializes a WalletBucket
// and sets up a unique index by name
func NewWalletBucket() WalletBucket {
	b := orm.NewBucket(BucketNameWallet, NewWallet(nil)).
		WithIndex(IndexName, nameIndex, true)
	return WalletBucket{Bucket: b}
}

// GetOrCreate will return the token if found, or create one
// with the given name otherwise.
func (b WalletBucket) GetOrCreate(db weave.KVStore, key weave.Address) (orm.Object, error) {
	obj, err := b.Get(db, key)
	if err == nil && obj == nil {
		obj = NewWallet(key)
	}
	return obj, err
}

// GetByName queries the wallet by secondary index on name,
// may return nil or a matching wallet
func (b WalletBucket) GetByName(db weave.KVStore, name string) (orm.Object, error) {
	objs, err := b.GetIndexed(db, IndexName, []byte(name))
	if err != nil {
		return nil, err
	}
	// objs may have 0 or 1 element (as index is unique)
	if len(objs) == 0 {
		return nil, nil
	}
	return objs[0], nil
}

// Save enforces the proper type
func (b WalletBucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*Wallet); !ok {
		return ErrInvalidObject(obj.Value())
	}
	return b.Bucket.Save(db, obj)
}

// simple indexer for Wallet name
func nameIndex(obj orm.Object) ([]byte, error) {
	if obj == nil {
		return nil, ErrInvalidIndex("nil")
	}
	wallet, ok := obj.Value().(*Wallet)
	if !ok {
		return nil, ErrInvalidIndex("Not wallet")
	}
	// big-endian encoded int64
	return []byte(wallet.Name), nil
}

// Named is any object that allows getting/setting a string name
// the object should be able to validate if SetName is a valid
type Named interface {
	GetName() string
	SetName(string) error
}

// NamedBucket is a bucket that can handle object with Get/SetName
// The object it returns must support AsNamed (only checked runtime :()
type NamedBucket interface {
	GetOrCreate(db weave.KVStore, key weave.Address) (orm.Object, error)
	Get(db weave.ReadOnlyKVStore, key []byte) (orm.Object, error)
	GetByName(db weave.KVStore, name string) (orm.Object, error)
	Save(db weave.KVStore, obj orm.Object) error
}
