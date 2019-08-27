package orm

import (
	"reflect"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x"
)

var _ x.Validater = (*SimpleObj)(nil)

// SimpleObj wraps a key and a value together
// It can be used as a template for type-safe objects
type SimpleObj struct {
	key   []byte
	value Model
}

// NewSimpleObj will combine a key and value into an object
func NewSimpleObj(key []byte, value Model) *SimpleObj {
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
		return errors.Field("Key", errors.ErrEmpty, "missing key")
	}
	if o.value == nil {
		return errors.Field("Value", errors.ErrEmpty, "missing value")
	}
	return errors.Field("Value", o.value.Validate(), "invalid value")
}

// SetKey may be used to update a simple obj key
func (o *SimpleObj) SetKey(key []byte) {
	o.key = key
}

// Clone will make a copy of this object
func (o *SimpleObj) Clone() Object {
	cpy := reflect.New(reflect.TypeOf(o.value).Elem()).Interface().(Model)
	res := &SimpleObj{
		value: cpy,
	}
	// only copy key if non-nil
	if len(o.key) > 0 {
		res.key = append([]byte(nil), o.key...)
	}
	return res
}
