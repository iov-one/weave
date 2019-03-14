package orm

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x"
)

var _ Object = (*SimpleObj)(nil)
var _ Cloneable = (*SimpleObj)(nil)
var _ x.Validater = (*SimpleObj)(nil)

// SimpleObj wraps a key and a value together
// It can be used as a template for type-safe objects
type SimpleObj struct {
	key   []byte
	value CloneableData
}

// NewSimpleObj will combine a key and value into an object
func NewSimpleObj(key []byte, value CloneableData) *SimpleObj {
	return &SimpleObj{
		key:   key,
		value: value,
	}
}

// Value gets the value stored in the object
func (o SimpleObj) Value() weave.Persistent {
	return o.value
}

// Key returns the key to store the object under
func (o SimpleObj) Key() []byte {
	return o.key
}

// Validate makes sure the fields aren't empty.
// And delegates to the value validator if present
func (o SimpleObj) Validate() error {
	if len(o.key) == 0 {
		return errors.Wrap(errors.ErrEmpty, "missing key")
	}
	if o.value == nil {
		return errors.Wrap(errors.ErrEmpty, "missing value")
	}
	return o.value.Validate()
}

// SetKey may be used to update a simple obj key
func (o *SimpleObj) SetKey(key []byte) {
	o.key = key
}

// Clone will make a copy of this object
func (o *SimpleObj) Clone() Object {
	res := &SimpleObj{
		value: o.value.Copy(),
	}
	// only copy key if non-nil
	if len(o.key) > 0 {
		res.key = append([]byte(nil), o.key...)
	}
	return res
}
