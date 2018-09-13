package multisig

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
)

const (
	// BucketName is where we store the contracts
	BucketName = "contracts"
	// SequenceName is an auto-increment ID counter for contracts
	SequenceName = "id"
)

// enforce that Contract fulfils desired interface compile-time
var _ orm.CloneableData = (*Contract)(nil)

// Validate enforces sigs and threshold boundaries
func (c *Contract) Validate() error {
	if len(c.Sigs) == 0 {
		return ErrMissingSigs()
	}
	if c.ActivationThreshold < 0 || int(c.ActivationThreshold) > len(c.Sigs) {
		return ErrInvalidActivationThreshold()
	}
	if c.ChangeThreshold < 0 {
		return ErrInvalidChangeThreshold()
	}
	for _, a := range c.Sigs {
		if err := weave.Address(a).Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Copy makes a new Profile with the same data
func (c *Contract) Copy() orm.CloneableData {
	return &Contract{
		Sigs:                c.Sigs,
		ActivationThreshold: c.ActivationThreshold,
		ChangeThreshold:     c.ChangeThreshold,
	}
}

// ContractBucket is a type-safe wrapper around orm.Bucket
type ContractBucket struct {
	orm.Bucket
	idSeq orm.Sequence
}

// NewContractBucket initializes a ContractBucket with default name
//
// inherit Get and Save from orm.Bucket
// add run-time check on Save
func NewContractBucket() ContractBucket {
	bucket := orm.NewBucket(BucketName,
		orm.NewSimpleObj(nil, new(Contract)))
	return ContractBucket{
		Bucket: bucket,
		idSeq:  bucket.Sequence(SequenceName),
	}
}

// Save enforces the proper type
func (b ContractBucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*Contract); !ok {
		return orm.ErrInvalidObject(obj.Value())
	}
	return b.Bucket.Save(db, obj)
}
