package aswap

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &Swap{}, migration.NoModification)
}

var _ orm.CloneableData = (*Swap)(nil)

// Validate ensures the Swap is valid
func (s *Swap) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", s.Metadata.Validate())
	errs = errors.AppendField(errs, "Source", s.Source.Validate())
	errs = errors.AppendField(errs, "Destination", s.Destination.Validate())
	if len(s.PreimageHash) != preimageHashSize {
		errs = errors.Append(errs, errors.Field("PreimageHash", errors.ErrInput, "preimage hash has to be exactly %d bytes", preimageHashSize))
	}
	if s.Timeout == 0 {
		// Zero timeout is a valid value that dates to 1970-01-01. We
		// know that this value is in the past and makes no sense. Most
		// likely value was not provided and a zero value remained.
		errs = errors.Append(errs, errors.Field("Timeout", errors.ErrInput, "timeout is required"))
	}
	errs = errors.AppendField(errs, "Timeout", s.Timeout.Validate())
	if len(s.Memo) > maxMemoSize {
		errs = errors.Append(errs, errors.Field("Memo", errors.ErrInput, "memo must be not longer than %d characters", maxMemoSize))
	}
	errs = errors.AppendField(errs, "Address", s.Address.Validate())
	return errs
}

// Copy makes a new swap
func (s *Swap) Copy() orm.CloneableData {
	return &Swap{
		Metadata:     s.Metadata.Copy(),
		PreimageHash: s.PreimageHash,
		Source:       s.Source,
		Destination:  s.Destination,
		Timeout:      s.Timeout,
		Memo:         s.Memo,
		Address:      s.Address.Clone(),
	}
}

// AsSwap extracts a *Swap value or nil from the object
// Must be called on a Bucket result that is an *Swap,
// will panic on bad type.
func AsSwap(obj orm.Object) *Swap {
	if obj == nil || obj.Value() == nil {
		return nil
	}
	return obj.Value().(*Swap)
}

func NewBucket() orm.ModelBucket {
	b := orm.NewModelBucket("swap", &Swap{},
		orm.WithIDSequence(swapSeq),
		orm.WithIndex("source", idxSource, false),
		orm.WithIndex("destination", idxDestination, false),
		orm.WithIndex("preimage_hash", idxPrehash, false),
	)
	return migration.NewModelBucket("aswap", b)
}

var swapSeq = orm.NewSequence("aswap", "id")

func toSwap(obj orm.Object) (*Swap, error) {
	if obj == nil {
		return nil, errors.Wrap(errors.ErrHuman, "Cannot take index of nil")
	}
	esc, ok := obj.Value().(*Swap)
	if !ok {
		return nil, errors.Wrap(errors.ErrHuman, "Can only take index of Swap")
	}
	return esc, nil
}

func idxSource(obj orm.Object) ([]byte, error) {
	swp, err := toSwap(obj)
	if err != nil {
		return nil, err
	}
	return swp.Source, nil
}

func idxDestination(obj orm.Object) ([]byte, error) {
	swp, err := toSwap(obj)
	if err != nil {
		return nil, err
	}
	return swp.Destination, nil
}

func idxPrehash(obj orm.Object) ([]byte, error) {
	swp, err := toSwap(obj)
	if err != nil {
		return nil, err
	}
	return swp.PreimageHash, nil
}
