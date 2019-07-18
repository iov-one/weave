package sigs

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &UserData{}, migration.NoModification)
}

// BucketName is where we store the accounts
const BucketName = "sigs"

//---- UserData
// Model stores the persistent state and all domain logic
// associated with valid state and state transitions.

var _ orm.CloneableData = (*UserData)(nil)

func (u *UserData) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", u.Metadata.Validate())
	if seq := u.Sequence; seq < 0 {
		errs = errors.AppendField(errs, "Sequence", ErrInvalidSequence)
	} else if seq > 0 && u.Pubkey == nil {
		errs = errors.Append(errs, errors.Field("Sequence", ErrInvalidSequence, "needs Pubkey"))
	}
	return errs
}

// Copy makes a new UserData with the same coins
func (u *UserData) Copy() orm.CloneableData {
	return &UserData{
		Metadata: u.Metadata.Copy(),
		Sequence: u.Sequence,
		Pubkey:   u.Pubkey,
	}
}

// CheckAndIncrementSequence implements check and increment operation.
// If current sequence value is the same as given expected value then it is
// incremented. Otherwise an error is returned.
// Before incrementing the sequence, this function is testing for a value
// overflow.
func (u *UserData) CheckAndIncrementSequence(expected int64) error {
	if u.Sequence != expected {
		return errors.Wrapf(ErrInvalidSequence, "mismatch expected %d, got %d", expected, u.Sequence)
	}

	next := u.Sequence + 1

	// maxSequenceValue is limited by the client. The greatest supported
	// nonce value at client side is
	//   Number.MAX_SAFE_INTEGER = 9007199254740991 = 2^53 −  1
	// If greater values must be supported, we get much more complicated
	// client code.
	const maxSequenceValue = (1 << 53) - 1
	if next <= 0 || next > maxSequenceValue {
		return errors.Wrap(errors.ErrOverflow, "sequence out of range")
	}
	u.Sequence = next
	return nil
}

// SetPubkey will try to set the Pubkey or panic on an illegal operation.
// It is illegal to reset an already set key
// Otherwise, we don't control
// (although we could verify the hash, we leave that to the controller)
func (u *UserData) SetPubkey(pubkey *crypto.PublicKey) {
	if u.Pubkey != nil {
		panic("Cannot change pubkey for a user")
	}
	u.Pubkey = pubkey
}

//-------------------- Object Wrapper -------

// AsUser will safely type-cast any value from Bucket to a UserData
func AsUser(obj orm.Object) *UserData {
	if obj == nil || obj.Value() == nil {
		return nil
	}
	return obj.Value().(*UserData)
}

// NewUser constructs an object from an address and pubkey
func NewUser(pubkey *crypto.PublicKey) orm.Object {
	var key weave.Address
	value := &UserData{
		Metadata: &weave.Metadata{Schema: 1},
		Pubkey:   pubkey,
	}
	if pubkey != nil {
		key = pubkey.Address()
	}
	return orm.NewSimpleObj(key, value)
}

// Bucket extends orm.Bucket with GetOrCreate
type Bucket struct {
	orm.Bucket
}

// NewBucket creates the proper bucket for this extension
func NewBucket() Bucket {
	return Bucket{
		Bucket: migration.NewBucket("sigs", BucketName, NewUser(nil)),
	}
}

// GetOrCreate initializes a UserData if none exist for that key
func (b Bucket) GetOrCreate(db weave.KVStore, pubkey *crypto.PublicKey) (orm.Object, error) {
	obj, err := b.Get(db, pubkey.Address())
	if err == nil && obj == nil {
		obj = NewUser(pubkey)
	}
	return obj, err
}
