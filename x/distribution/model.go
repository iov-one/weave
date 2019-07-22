package distribution

import (
	"math"

	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &Revenue{}, migration.NoModification)
}

var _ orm.CloneableData = (*Revenue)(nil)

func (rev *Revenue) Validate() error {
	var errs error

	errs = errors.AppendField(errs, "Metadata", rev.Metadata.Validate())
	errs = errors.AppendField(errs, "Admin", rev.Admin.Validate())
	errs = errors.AppendField(errs, "Destinatinos", validateDestinations(rev.Destinations, errors.ErrModel))
	errs = errors.AppendField(errs, "Address", rev.Address.Validate())

	return errs
}

// validateDestinations returns an error if given list of destinations is not
// valid. This functionality is used in many places (model and messages),
// having it abstracted saves repeating validation code.
// Model validation returns different class of error than message validation,
// that is why require base error class to be given.
func validateDestinations(rs []*Destination, baseErr *errors.Error) error {
	var errs error

	switch n := len(rs); {
	case n == 0:
		errs = errors.Append(errs, errors.Wrap(baseErr, "no destinations"))
	case n > maxDestinations:
		errs = errors.Append(errs, errors.Wrap(baseErr, "too many destinations"))
	}

	// Destination address must not repeat. Repeating addresses would not
	// cause an issue, but requiring them to be unique increase
	// configuration clarity.
	addresses := make(map[string]struct{})

	for i, r := range rs {
		switch {
		case r.Weight <= 0:
			errs = errors.Append(errs, errors.Wrapf(baseErr, "destination %d invalid weight", i))
		case r.Weight > maxWeight:
			errs = errors.Append(errs, errors.Wrapf(baseErr, "weight must not be greater than %d", maxWeight))
		}

		if err := r.Address.Validate(); err != nil {
			errs = errors.Append(errs, errors.Wrapf(err, "destination %d address", i))
		}
		addr := r.Address.String()
		if _, ok := addresses[addr]; ok {
			errs = errors.Append(errs, errors.Wrapf(baseErr, "address %q is not unique", addr))
		}
		addresses[addr] = struct{}{}

	}

	return errs
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
		Admin:        rev.Admin.Clone(),
		Destinations: make([]*Destination, len(rev.Destinations)),
		Address:      rev.Address.Clone(),
	}
	for i := range rev.Destinations {
		cpy.Destinations[i] = &Destination{
			Address: rev.Destinations[i].Address.Clone(),
			Weight:  rev.Destinations[i].Weight,
		}
	}
	return cpy
}

// NewRevenueBucket returns a bucket for managing revenues state.
func NewRevenueBucket() orm.ModelBucket {
	b := orm.NewModelBucket("revenue", &Revenue{},
		orm.WithIDSequence(revenueSeq),
	)
	return migration.NewModelBucket("distribution", b)
}

var revenueSeq = orm.NewSequence("revenue", "id")

func RevenueAccount(key []byte) weave.Address {
	return weave.NewCondition("dist", "revenue", key).Address()
}
