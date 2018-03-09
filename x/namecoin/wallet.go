package namecoin

import (
	"github.com/confio/weave"
	"github.com/confio/weave/orm"
	"github.com/confio/weave/x"
)

const (
	// BucketNameWallet is where we store the balances
	BucketNameWallet = "wllt"
	// IndexName is the index to query wallet by name
	IndexName = "name"
)

//--- Wallet

var _ orm.CloneableData = (*Wallet)(nil)

func (w *Wallet) xcoins() x.Coins {
	return x.Coins(w.GetCoins())
}

// Validate requires that all coins are in alphabetical
func (w *Wallet) Validate() error {
	return w.xcoins().Validate()
}

// Copy makes a new set with the same coins
func (w *Wallet) Copy() orm.CloneableData {
	return &Wallet{
		Name:  w.Name,
		Coins: w.xcoins().Clone(),
	}
}

// AsWallet safely extracts a Wallet value from the object
func AsWallet(obj orm.Object) *Wallet {
	if obj == nil || obj.Value() == nil {
		return nil
	}
	return obj.Value().(*Wallet)
}

// NewWallet creates an empty wallet with this address
// serves as an object for the bucket
func NewWallet(key weave.Address) orm.Object {
	return orm.NewSimpleObj(key, new(Wallet))
}

//--- WalletBucket - handles tokens

// WalletBucket is a type-safe wrapper around orm.Bucket
type WalletBucket struct {
	orm.Bucket
}

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
