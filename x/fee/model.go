package fee

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

var _ orm.CloneableData = (*TransactionFee)(nil)

// NewTokenInfo returns a new TransactionFee in orm representation
func NewTransactionFee(id string, fee x.Coin) orm.Object {
	return orm.NewSimpleObj([]byte(id), &TransactionFee{
		Fee: &fee,
	})
}

func (t *TransactionFee) Validate() error {
	err := t.Fee.Validate()
	if err != nil {
		return err
	}
	return nil
}

func (t *TransactionFee) Copy() orm.CloneableData {
	return &TransactionFee{
		Fee: t.Fee,
	}
}

// TransactionFeeBucket stores TransactionFee, where key is a combination of a transaction message path
// Path() -> e.g. nft/approval/add for static fees and a Path() + namespace for dynamic fees, e.g.
// nft/approval/add:johndoenft
type TransactionFeeBucket struct {
	orm.Bucket
}

func NewTransactionFeeBucket() *TransactionFeeBucket {
	return &TransactionFeeBucket{
		Bucket: orm.NewBucket("fee", orm.NewSimpleObj(nil, &TransactionFee{})),
	}
}

func (b *TransactionFeeBucket) Get(db weave.KVStore, key string) (orm.Object, error) {
	return b.Bucket.Get(db, []byte(key))
}

func (b *TransactionFeeBucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*TransactionFee); !ok {
		return orm.ErrInvalidObject(obj.Value())
	}
	return b.Bucket.Save(db, obj)
}
