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

// Validate requires that all coins are in alphabetical
func (u *UserData) Validate() error {
	if err := u.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	seq := u.Sequence
	if seq < 0 {
		return errors.Wrapf(ErrInvalidSequence, "Seq(%d)", seq)
	}
	if seq > 0 && u.Pubkey == nil {
		return errors.Wrapf(ErrInvalidSequence, "Seq(%d) needs Pubkey", seq)
	}
	return nil
}

// Copy makes a new UserData with the same coins
func (u *UserData) Copy() orm.CloneableData {
	return &UserData{
		Metadata: u.Metadata.Copy(),
		Sequence: u.Sequence,
		Pubkey:   u.Pubkey,
	}
}

// CheckAndIncrementSequence checks if the current Sequence
// matches the expected value.
// If so, it will increase the sequence by one and return nil
// If not, it will not change the sequence, but return an error
func (u *UserData) CheckAndIncrementSequence(check int64) error {
	if u.Sequence != check {
		return errors.Wrapf(ErrInvalidSequence, "mismatch expected %d, got %d", check, u.Sequence)
	}
	u.Sequence++
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

//------------------ High-Level ------------------------

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
