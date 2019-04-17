package gconf

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

// Store is a subset of weave.KVStore
type Store interface {
	Get([]byte) []byte
	Set([]byte, []byte)
}

// Save will Validate the object, before writing it to a special "configuration"
// singleton for that package name.
func Save(db Store, pkg string, src ValidMarshaler) error {
	key := []byte("_c:" + pkg)
	if err := src.Validate(); err != nil {
		return errors.Wrap(err, "saving gconf")
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
		return errors.Wrapf(err, "unmarshal: key %q", key)
	}
	return nil
}

// Unmarshaler is implemented by object that can load their state from given
// binary representation. This interface is implemented by all protobuf
// messages.
type Unmarshaler interface {
	Unmarshal([]byte) error
}

type Configuration interface {
	ValidMarshaler
	Unmarshaler
}

// InitConfig will take opts["conf"][pkg], parse it into the given Configuration object
// validate it, and store under the proper key in the database
// Returns an error if anything goes wrong
func InitConfig(db Store, opts weave.Options, pkg string, conf Configuration) error {
	var confOptions weave.Options
	if err := opts.ReadOptions("conf", &confOptions); err != nil {
		return errors.Wrap(err, "read conf")
	}
	if confOptions[pkg] == nil {
		return errors.Wrapf(errors.ErrInvalidInput, "no configuration for %s", pkg)
	}
	if err := confOptions.ReadOptions(pkg, conf); err != nil {
		return errors.Wrapf(err, "read configuration for %s", pkg)
	}
	if err := Save(db, pkg, conf); err != nil {
		return errors.Wrapf(err, "save configuration for %s", pkg)
	}
	return nil
}
