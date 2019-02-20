package distribution

import (
	"fmt"
	"math"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
)

var _ orm.CloneableData = (*Revenue)(nil)

func (rev *Revenue) Validate() error {
	if err := rev.Admin.Validate(); err != nil {
		return errors.Wrap(err, "invalid admin signature")
	}
	if err := validateRecipients(rev.Recipients, errors.ErrInvalidModel); err != nil {
		return err
	}
	return nil
}

// validateRecipients returns an error if given list of recipients is not
// valid. This functionality is used in many places (model and messages),
// having it abstracted saves repeating validation code.
// Model validation returns different class of error than message validation,
// that is why require base error class to be given.
func validateRecipients(rs []*Recipient, baseErr errors.Error) error {
	switch n := len(rs); {
	case n == 0:
		return baseErr.New("no recipients")
	case n > maxRecipients:
		return baseErr.New("too many recipients")
	}

	// Recipient address must not repeat. Repeating addresses would not
	// cause an issue, but requiring them to be unique increase
	// configuration clarity.
	addresses := make(map[string]struct{})

	for i, r := range rs {
		switch {
		case r.Weight <= 0:
			return baseErr.New(fmt.Sprintf("recipient %d invalid weight", i))
		case r.Weight > maxWeight:
			return baseErr.New(fmt.Sprintf("weight must not be greater than %d", maxWeight))
		}

		if err := r.Address.Validate(); err != nil {
			return errors.Wrap(err, fmt.Sprintf("recipient %d address", i))
		}
		addr := r.Address.String()
		if _, ok := addresses[addr]; ok {
			return baseErr.New(fmt.Sprintf("address %q is not unique", addr))
		}
		addresses[addr] = struct{}{}

	}

	return nil
}

const (
	// maxRecipients defines the maximum number of recipients allowed within a
	// single revenue. This is a high number that should not be an issue in real
	// life scenarios. But having a sane limit allows us to avoid attacks.
	maxRecipients = 200

	// maxWeight defines the maximum value for the recipient weight. This
	// is a high number that for all recipient of a given revenue, when
	// combined does not exceed int32 capacity.
	maxWeight = math.MaxInt32 / (maxRecipients + 1)
)

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
	return weave.NewCondition("distribution", "revenue", revenueID).Address()
}

// Save persists the state of a given revenue entity.
func (b *RevenueBucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*Revenue); !ok {
		return errors.ErrInvalidModel.New(fmt.Sprintf("invalid type: %T", obj.Value()))
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
		return nil, errors.ErrNotFound.New("no revenue")
	}
	rev, ok := obj.Value().(*Revenue)
	if !ok {
		return nil, errors.ErrInvalidModel.New(fmt.Sprintf("invalid type: %T", obj.Value()))
	}
	return rev, nil
}
