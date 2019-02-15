package feedist

import (
	"fmt"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
)

var _ orm.CloneableData = (*Revenue)(nil)

func (rev *Revenue) Validate() error {
	if err := rev.Admin.Validate(); err != nil {
		return errors.Wrap(err, "invalid admin signature")
	}
	if len(rev.Recipients) == 0 {
		return errors.InvalidModelErr.New("no recipients")
	}
	for i, r := range rev.Recipients {
		if err := r.Address.Validate(); err != nil {
			return errors.Wrap(err, fmt.Sprintf("recipient %d address", i))
		}
		if r.Weight <= 0 {
			return errors.InvalidModelErr.New(fmt.Sprintf("recipient %d invalid weight", i))
		}
	}
	return nil
}

func (rev *Revenue) Copy() orm.CloneableData {
	cpy := &Revenue{
		Admin:      copyAddr(rev.Admin),
		Recipients: make([]*Recipient, len(rev.Recipients)),
	}
	for i := range rev.Recipients {
		cpy.Recipients[i] = &Recipient{
			Address: copyAddr(rev.Recipients[i].Address),
			Weight:  rev.Recipients[i].Weight,
		}
	}
	return cpy
}

func copyAddr(a weave.Address) weave.Address {
	cpy := make(weave.Address, len(a))
	copy(cpy, a)
	return cpy
}

type RevenueBucket struct {
	orm.Bucket
	idSeq orm.Sequence
}

// NewRevenueBucket returns a bucket for managing revenues state.
func NewRevenueBucket() *RevenueBucket {
	b := orm.NewBucket("revenue", orm.NewSimpleObj(nil, &Revenue{}))
	return &RevenueBucket{
		Bucket: b,
		idSeq:  b.Sequence("id"),
	}
}

// Create adds given revenue instance to the store and returns the ID of the
// newly inserted entity.
func (b *RevenueBucket) Create(db weave.KVStore, rev *Revenue) (orm.Object, error) {
	key := b.idSeq.NextVal(db)
	obj := orm.NewSimpleObj(key, rev)
	return obj, b.Bucket.Save(db, obj)
}

// RevenueAccount returns an account address that is holding funds of a revenue
// with given ID.
func RevenueAccount(revenueID []byte) weave.Address {
	return weave.NewCondition("feedist", "revenue", revenueID).Address()
}

// Save persists the state of a given revenue entity.
func (b *RevenueBucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*Revenue); !ok {
		return errors.InvalidModelErr.New(fmt.Sprintf("invalid type: %T", obj.Value()))
	}
	return b.Bucket.Save(db, obj)
}

// GetRevenue returns a revenue instance with given ID.
func (b *RevenueBucket) GetRevenue(db weave.KVStore, revenueID []byte) (*Revenue, error) {
	obj, err := b.Get(db, revenueID)
	if err != nil {
		return nil, errors.Wrap(err, "no revenue")
	}
	if obj == nil || obj.Value() == nil {
		return nil, errors.NotFoundErr.New("no revenue")
	}
	rev, ok := obj.Value().(*Revenue)
	if !ok {
		return nil, errors.InvalidModelErr.New(fmt.Sprintf("invalid type: %T", obj.Value()))
	}
	return rev, nil
}
