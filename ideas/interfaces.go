package ideas

import (
	"github.com/confio/weave"
)

// Object is what is stored in the bucket
// Key is joined with the prefix to set the full key
// Value is the data stored
//
// this can be light wrapper around a protobuf-defined type
type Object interface {
	Keyed
	Value() weave.Persistent

	// Validate returns error if the object is not in a valid
	// state to save to the db (eg. field missing, out of range, ...)
	Validate() error
}

// Keyed is anything that can identify itself
type Keyed interface {
	GetKey() []byte
}

// SetKeyer allows you to optionally change the key
type SetKeyer interface {
	SetKey([]byte)
}

// Cloneable will create a new object that can be loaded into
type Cloneable interface {
	Clone() Object
}
