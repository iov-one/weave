package orm

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
)

// Object is what is stored in the bucket
// Key is joined with the prefix to set the full key
// Value is the data stored
//
// this can be light wrapper around a protobuf-defined type
type Object interface {
	Keyed
	Cloneable
	// Validate returns error if the object is not in a valid
	// state to save to the db (eg. field missing, out of range, ...)
	x.Validater
	Value() weave.Persistent
}

// Reader defines an interface that allows reading objects from the db
type Reader interface {
	Get(db weave.ReadOnlyKVStore, key []byte) (Object, error)
}

// Keyed is anything that can identify itself
type Keyed interface {
	Key() []byte
	SetKey([]byte)
}

// Cloneable will create a new object that can be loaded into
type Cloneable interface {
	Clone() Object
}

// CloneableData is an intelligent Value that can be embedded
// in a simple object to handle much of the details.
type CloneableData interface {
	x.Validater
	weave.Persistent
	Copy() CloneableData
}
