package gconf

import (
	"reflect"

	"github.com/iov-one/weave/errors"
)

type Store interface {
	Get([]byte) []byte
	Set([]byte, []byte)
}

func Save(db Store, src Marshaler) error {
	key := []byte("configuration:" + pkgPath(src))
	raw, err := src.Marshal()
	if err != nil {
		return errors.Wrapf(err, "marshal: key %q", key)
	}
	db.Set(key, raw)
	return nil
}

// Marshaler is implemented by object that can serialize itself to a binary
// representation. This interface is implemented by all protobuf messages.
type Marshaler interface {
	Marshal() ([]byte, error)
}

func Load(db Store, dst Unmarshaler) error {
	key := []byte("configuration:" + pkgPath(dst))
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

// pkgPath returns the full package path that given structure belongs to. It
// returns an empty string of non structure types.
// Use full path instead of just the package name to avoid name collisions.
// Each package is expected to have only one configuration object.
func pkgPath(structure interface{}) string {
	t := reflect.TypeOf(structure)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return ""
	}
	return t.PkgPath()
}
