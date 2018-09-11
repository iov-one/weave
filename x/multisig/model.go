package multisig

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
)

// enforce that Contract fulfils desired interface compile-time
var _ orm.CloneableData = (*Contract)(nil)

// Validate enforces limits of text and title size
func (c *Contract) Validate() error {
	return nil
}

// Copy makes a new Profile with the same data
func (c *Contract) Copy() orm.CloneableData {
	return &Contract{
		Address:             c.Address,
		Sigs:                c.Sigs,
		ActivationThreshold: c.ActivationThreshold,
		ChangeThreshold:     c.ChangeThreshold,
	}
}

const ContractBucketName = "contracts"

// ContractBucket is a type-safe wrapper around orm.Bucket
type ContractBucket struct {
	orm.Bucket
}

// NewContractBucket initializes a ContractBucket with default name
//
// inherit Get and Save from orm.Bucket
// add run-time check on Save
func NewContractBucket() ContractBucket {
	bucket := orm.NewBucket(ContractBucketName,
		orm.NewSimpleObj(nil, new(Contract)))
	return ContractBucket{
		Bucket: bucket,
	}
}

// Save enforces the proper type
func (b ContractBucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*Contract); !ok {
		return orm.ErrInvalidObject(obj.Value())
	}
	return b.Bucket.Save(db, obj)
}
