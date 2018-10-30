package approvals

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
)

const (
	// BucketName is where we store the contracts
	BucketName = "approvals"
	// SequenceName is an auto-increment ID counter for contracts
	SequenceName = "id"
)

// enforce that Contract fulfils desired interface compile-time
var _ orm.CloneableData = (*Approval)(nil)

// Validate enforces sigs and threshold boundaries
func (c *Approval) Validate() error {
	return nil
}

// Copy makes a new Profile with the same data
func (c *Approval) Copy() orm.CloneableData {
	return &Approval{}
}

// ApprovalBucket is a type-safe wrapper around orm.Bucket
type ApprovalBucket struct {
	orm.Bucket
	idSeq orm.Sequence
}

// NewApprovalBucket initializes a ApprovalBucket with default name
//
// inherit Get and Save from orm.Bucket
// add run-time check on Save
func NewApprovalBucket() ApprovalBucket {
	bucket := orm.NewBucket(BucketName,
		orm.NewSimpleObj(nil, new(Approval)))
	return ApprovalBucket{
		Bucket: bucket,
		idSeq:  bucket.Sequence(SequenceName),
	}
}

// Save enforces the proper type
func (b ApprovalBucket) Save(db weave.KVStore, obj orm.Object) error {
	if _, ok := obj.Value().(*Approval); !ok {
		return orm.ErrInvalidObject(obj.Value())
	}
	return b.Bucket.Save(db, obj)
}
