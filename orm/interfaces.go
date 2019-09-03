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
	// Validate returns error if the object is not in a valid
	// state to save to the db (eg. field missing, out of range, ...)
	x.Validater
	Value() weave.Persistent

	Key() []byte
	SetKey([]byte)
}

// CloneableData is an intelligent Value that can be embedded
// in a simple object to handle much of the details.
//
// CloneableData interface is deprecated and must not be used anymore.
type CloneableData interface {
	x.Validater
	weave.Persistent
}
