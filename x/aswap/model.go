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
	if err := s.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if err := s.Source.Validate(); err != nil {
		return errors.Wrap(err, "source")
	}
	if err := s.Destination.Validate(); err != nil {
		return errors.Wrap(err, "destination")
	}
	if len(s.PreimageHash) != preimageHashSize {
		return errors.Wrapf(errors.ErrInput,
			"preimage hash has to be exactly %d bytes", preimageHashSize)
	}
	if s.Timeout == 0 {
		// Zero timeout is a valid value that dates to 1970-01-01. We
		// know that this value is in the past and makes no sense. Most
		// likely value was not provided and a zero value remained.
		return errors.Wrap(errors.ErrInput, "timeout is required")
	}
	if err := s.Timeout.Validate(); err != nil {
		return errors.Wrap(err, "invalid timeout value")
	}
	if len(s.Memo) > maxMemoSize {
		return errors.Wrapf(errors.ErrInput, "memo %s", s.Memo)
	}
	if err := s.Address.Validate(); err != nil {
		return errors.Wrap(err, "address")
	}
	return nil
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
