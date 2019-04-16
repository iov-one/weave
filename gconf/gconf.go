package gconf

import (
	"github.com/iov-one/weave/errors"
)

// Store is a subset of weave.KVStore
type Store interface {
	Get([]byte) []byte
	Set([]byte, []byte)
}

// Save will validate code
func Save(db Store, pkg string, src ValidMarshaler) error {
	key := []byte("_c:" + pkg)
	if err := src.Validate(); err != nil {
		return errors.Wrap(err, "Saving gconf")
	}
	raw, err := src.Marshal()
	if err != nil {
		return errors.Wrapf(err, "marshal: key %q", key)
	}
	db.Set(key, raw)
	return nil
}

// ValidMarshaler is implemented by object that can serialize itself to a binary
// representation. Marshal is implemented by all protobuf messages.
// You must add your own Validate method
//
// Note duplicate of code in x/persistent.go
type ValidMarshaler interface {
	Marshal() ([]byte, error)
	Validate() error
}

func Load(db Store, pkg string, dst Unmarshaler) error {
	key := []byte("_c:" + pkg)
	raw := db.Get(key)
	if raw == nil {
		return errors.Wrapf(errors.ErrNotFound, "key %q", key)
	}
	if err := dst.Unmarshal(raw); err != nil {
		return errors.Wrapf(err, "unmarhsal: key %q", key)
	}
	return nil
}

// Unmarshaler is implemented by object that can load their state from given
// binary representation. This interface is implemented by all protobuf
// messages.
type Unmarshaler interface {
	Unmarshal([]byte) error
}
