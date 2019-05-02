package aswap

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &Swap{}, migration.NoModification)
}

const (
	// BucketName is where we store the swaps
	BucketName = "aswp"
	// SequenceName is an auto-increment ID counter for swaps
	SequenceName = "id"
)

var _ orm.CloneableData = (*Swap)(nil)

// Validate ensures the Swap is valid
func (s *Swap) Validate() error {
	if err := s.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if err := s.Src.Validate(); err != nil {
		return errors.Wrap(err, "src")
	}
	if err := validatePreimageHash(s.PreimageHash); err != nil {
		return err
	}
	if err := s.Recipient.Validate(); err != nil {
		return errors.Wrap(err, "recipient")
	}
	if s.Timeout == 0 {
		// Zero timeout is a valid value that dates to 1970-01-01. We
		// know that this value is in the past and makes no sense. Most
		// likely value was not provided and a zero value remained.
		return errors.Wrap(errors.ErrInvalidInput, "timeout is required")
	}
	if err := s.Timeout.Validate(); err != nil {
		return errors.Wrap(err, "invalid timeout value")
	}
	if len(s.Memo) > maxMemoSize {
		return errors.Wrapf(errors.ErrInvalidInput, "memo %s", s.Memo)
	}
	return nil
}

// Copy makes a new swap with the same coins
func (s *Swap) Copy() orm.CloneableData {
	return &Swap{
		Metadata:     s.Metadata.Copy(),
		Src:          s.Src,
		PreimageHash: s.PreimageHash,
		Recipient:    s.Recipient,
		Timeout:      s.Timeout,
		Memo:         s.Memo,
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

// Condition calculates the address of a swap given
// the key
func Condition(key []byte) weave.Condition {
	return weave.NewCondition("aswap", "seq", key)
}

// Bucket is a type-safe wrapper around orm.Bucket
type Bucket struct {
	orm.Bucket
	idSeq orm.Sequence
}

// NewBucket initializes a Bucket with default name
//
// inherit Get and Save from orm.Bucket
// add Create
func NewBucket() Bucket {
	bucket := migration.NewBucket("aswap", BucketName,
		orm.NewSimpleObj(nil, &Swap{})).
		WithIndex("src", idxSrc, false).
		WithIndex("recipient", idxRecipient, false).
		WithIndex("preimage_hash", idxPreimageHash, false)

	return Bucket{
		Bucket: bucket,
		idSeq:  bucket.Sequence(SequenceName),
	}
}

func getSwap(obj orm.Object) (*Swap, error) {
	if obj == nil {
		return nil, errors.Wrap(errors.ErrHuman, "Cannot take index of nil")
	}

	swp, ok := obj.Value().(*Swap)
	if !ok {
		return nil, errors.Wrap(errors.ErrHuman, "Can only take index of Swap")
	}
	return swp, nil
}

func idxSrc(obj orm.Object) ([]byte, error) {
	swp, err := getSwap(obj)
	if err != nil {
		return nil, err
	}
	return swp.Src, nil
}

func idxRecipient(obj orm.Object) ([]byte, error) {
	swp, err := getSwap(obj)
	if err != nil {
		return nil, err
	}
	return swp.Recipient, nil
}

func idxPreimageHash(obj orm.Object) ([]byte, error) {
	swp, err := getSwap(obj)
	if err != nil {
		return nil, err
	}
	return swp.PreimageHash, nil
}

// Build assigns an ID to given swap instance and returns it as an orm
// Object. It does not persist the swap in the store.
func (b Bucket) Build(db weave.KVStore, swap *Swap) (orm.Object, error) {
	key, err := b.idSeq.NextVal(db)
	if err != nil {
		return nil, err
	}
	return orm.NewSimpleObj(key, swap), nil
}

// Save enforces the proper type
func (b Bucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*Swap); !ok {
		return errors.WithType(errors.ErrInvalidModel, obj.Value())
	}
	return b.Bucket.Save(db, obj)
}
