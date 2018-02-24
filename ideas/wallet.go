package ideas

import (
	"errors"

	"github.com/confio/weave"
)

const walletSeqID = "id"

// WalletBucket is a type-safe bucket to store wallets
type WalletBucket struct {
	bucket Bucket
}

func NewWalletBucket() WalletBucket {
	// TODO
	empty := new(WalletObj)
	create := new(WalletObj)
	return WalletBucket{
		bucket: NewBucket("wllt", empty, create),
	}
}

func (b WalletBucket) Create(key []byte) *WalletObj {
	return b.bucket.Create(key).(*WalletObj)
}

func (b WalletBucket) Get(db weave.KVStore, key []byte) (*WalletObj, error) {
	obj, err := b.bucket.Get(db, key)
	if obj != nil {
		return obj.(*WalletObj), nil
	}
	return nil, err
}

func (b WalletBucket) GetOrCreate(db weave.KVStore, key []byte) (*WalletObj, error) {
	obj, err := b.bucket.GetOrCreate(db, key)
	if obj != nil {
		return obj.(*WalletObj), nil
	}
	return nil, err
}

func (b WalletBucket) Save(db weave.KVStore, wallet *WalletObj) error {
	return b.bucket.Save(db, wallet)
}

// NextID returns next value from a sequence
func (b WalletBucket) NextWalletID(db weave.KVStore) int64 {
	seq := b.bucket.Sequence(walletSeqID)
	return seq.NextInt(db)
}

//----------- Wallet ---------
// Wrap wallet

var _ Object = (*WalletObj)(nil)
var _ Cloneable = (*WalletObj)(nil)
var _ SetKeyer = (*WalletObj)(nil)

// WalletObj wraps a wallet with key info
type WalletObj struct {
	key    []byte
	wallet *Wallet
}

func (w WalletObj) Value() weave.Persistent {
	return w.wallet
}

func (w WalletObj) GetKey() []byte {
	return w.key
}

func (w WalletObj) Validate() error {
	if len(w.key) == 0 {
		// TODO: enforce length here???
		return errors.New("Missing key")
	}
	if w.wallet == nil {
		return errors.New("Missing value")
	}
	// TODO: let wallet validate content
	return nil
}

func (w *WalletObj) SetKey(key []byte) {
	w.key = key
}

func (w *WalletObj) Clone() Object {
	wallet := new(Wallet)
	if w.wallet != nil {
		// clone the wallet
		coins := make([]*Coin, len(w.wallet.Coins))
		for i, c := range w.wallet.Coins {
			coins[i] = &Coin{
				Amount:       c.Amount,
				CurrencyCode: c.CurrencyCode,
			}
		}
		wallet = &Wallet{Coins: coins}
	}
	return WalletObj{
		wallet: wallet,
	}
}
