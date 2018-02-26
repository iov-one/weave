package x

//--------------- serialization stuff ---------------------

// Marshaller is anything that can be represented in binary
//
// Marshall may validate the data before serializing it and
// unless you previously validated the struct,
// errors should be expected.
type Marshaller interface {
	Marshal() ([]byte, error)
}

// Persistent supports Marshal and Unmarshal
//
// This is separated from Marshal, as this almost always requires
// a pointer, and functions that only need to marshal bytes can
// use the Marshaller interface to access non-pointers.
//
// As with Marshaller, this may do internal validation on the data
// and errors should be expected.
type Persistent interface {
	Marshaller
	Unmarshal([]byte) error
}

// MustMarshal will succeed or panic
func MustMarshal(obj Marshaller) []byte {
	bz, err := obj.Marshal()
	if err != nil {
		panic(err)
	}
	return bz
}

// MustUnmarshal will succeed or panic
func MustUnmarshal(obj Persistent, bz []byte) {
	err := obj.Unmarshal(bz)
	if err != nil {
		panic(err)
	}
}

//-------------------- Validation ---------

// Validater is any struct that can be validated.
// Not the same as a Validator, which votes on the blocks.
type Validater interface {
	Validate() error
}

// MustValidate panics if the object is not valid
func MustValidate(obj Validater) {
	err := obj.Validate()
	if err != nil {
		panic(err)
	}
}

// MarshalValidater is something that can be validated and
// serialized
type MarshalValidater interface {
	Marshaller
	Validater
}

// MustMarshalValid marshals the object, but panics
// if the object is not valid or has trouble marshalling
func MustMarshalValid(obj MarshalValidater) []byte {
	MustValidate(obj)
	return MustMarshal(obj)
}
