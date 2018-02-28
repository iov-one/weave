package x

import "github.com/confio/weave"

//--------------- serialization stuff ---------------------

// MustMarshal will succeed or panic
func MustMarshal(obj weave.Marshaller) []byte {
	bz, err := obj.Marshal()
	if err != nil {
		panic(err)
	}
	return bz
}

// MustUnmarshal will succeed or panic
func MustUnmarshal(obj weave.Persistent, bz []byte) {
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
	weave.Marshaller
	Validater
}

// MarshalValid validates the object, then marshals
func MarshalValid(obj MarshalValidater) ([]byte, error) {
	err := obj.Validate()
	if err != nil {
		return nil, err
	}
	return obj.Marshal()
}

// MustMarshalValid marshals the object, but panics
// if the object is not valid or has trouble marshalling
func MustMarshalValid(obj MarshalValidater) []byte {
	bz, err := MarshalValid(obj)
	if err != nil {
		panic(err)
	}
	return bz
}
