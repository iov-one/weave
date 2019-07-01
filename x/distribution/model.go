package distribution

import (
	"math"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &Revenue{}, migration.NoModification)
}

var _ orm.CloneableData = (*Revenue)(nil)

func (rev *Revenue) Validate() error {
	if err := rev.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "invalid metadata")
	}
	if err := rev.Admin.Validate(); err != nil {
		return errors.Wrap(err, "invalid admin signature")
	}
	if err := validateDestinations(rev.Destinations, errors.ErrModel); err != nil {
		return err
	}
	return nil
}

// validateDestinations returns an error if given list of destinations is not
// valid. This functionality is used in many places (model and messages),
// having it abstracted saves repeating validation code.
// Model validation returns different class of error than message validation,
// that is why require base error class to be given.
func validateDestinations(rs []*Destination, baseErr *errors.Error) error {
	switch n := len(rs); {
	case n == 0:
		return errors.Wrap(baseErr, "no destinations")
	case n > maxDestinations:
		return errors.Wrap(baseErr, "too many destinations")
	}

	// Destination address must not repeat. Repeating addresses would not
	// cause an issue, but requiring them to be unique increase
	// configuration clarity.
	addresses := make(map[string]struct{})

	for i, r := range rs {
		switch {
		case r.Weight <= 0:
			return errors.Wrapf(baseErr, "destination %d invalid weight", i)
		case r.Weight > maxWeight:
			return errors.Wrapf(baseErr, "weight must not be greater than %d", maxWeight)
		}

		if err := r.Address.Validate(); err != nil {
			return errors.Wrapf(err, "destination %d address", i)
		}
		addr := r.Address.String()
		if _, ok := addresses[addr]; ok {
			return errors.Wrapf(baseErr, "address %q is not unique", addr)
		}
		addresses[addr] = struct{}{}

	}

	return nil
}

const (
	// maxDestinations defines the maximum number of destinations allowed within a
	// single revenue. This is a high number that should not be an issue in real
	// life scenarios. But having a sane limit allows us to avoid attacks.
	maxDestinations = 200

	// maxWeight defines the maximum value for the destination weight. This
	// is a high number that for all destination of a given revenue, when
	// combined does not exceed int32 capacity.
	maxWeight = math.MaxInt32 / (maxDestinations + 1)
)

func (rev *Revenue) Copy() orm.CloneableData {
	cpy := &Revenue{
		Metadata:     rev.Metadata.Copy(),
		Admin:        copyAddr(rev.Admin),
		Destinations: make([]*Destination, len(rev.Destinations)),
	}
	for i := range rev.Destinations {
		cpy.Destinations[i] = &Destination{
			Address: copyAddr(rev.Destinations[i].Address),
			Weight:  rev.Destinations[i].Weight,
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
	orm.IDGenBucket
}

// NewRevenueBucket returns a bucket for managing revenues state.
func NewRevenueBucket() *RevenueBucket {
	b := migration.NewBucket("distribution", "revenue", orm.NewSimpleObj(nil, &Revenue{}))
	return &RevenueBucket{
		IDGenBucket: orm.WithSeqIDGenerator(b, "id"),
	}
}

// RevenueAccount returns an account address that is holding funds of a revenue
// with given ID.
func RevenueAccount(revenueID []byte) (weave.Address, error) {
	c := weave.NewCondition("dist", "revenue", revenueID)
	if err := c.Validate(); err != nil {
		return nil, errors.Wrap(err, "condition")
	}
	return c.Address(), nil
}

// Save persists the state of a given revenue entity.
func (b *RevenueBucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*Revenue); !ok {
		return errors.Wrapf(errors.ErrModel, "invalid type: %T", obj.Value())
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
		return nil, errors.Wrap(errors.ErrNotFound, "no revenue")
	}
	rev, ok := obj.Value().(*Revenue)
	if !ok {
		return nil, errors.Wrapf(errors.ErrModel, "invalid type: %T", obj.Value())
	}
	return rev, nil
}
