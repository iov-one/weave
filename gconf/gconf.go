package gconf

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
)

// ReadStore is a subset of weave.ReadOnlyKVStore.
type ReadStore interface {
	Get([]byte) ([]byte, error)
}

// Store is a subset of weave.KVStore.
type Store interface {
	ReadStore
	Set([]byte, []byte) error
}

// Save will Validate the object, before writing it to a special "configuration"
// singleton for that package name.
func Save(db Store, pkg string, src ValidMarshaler) error {
	key := dbkey(pkg)
	if err := src.Validate(); err != nil {
		return errors.Wrapf(err, "validation: key %q", key)
	}
	raw, err := src.Marshal()
	if err != nil {
		return errors.Wrapf(err, "marshal: key %q", key)
	}
	return db.Set(key, raw)
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

func Load(db ReadStore, pkg string, dst Unmarshaler) error {
	key := dbkey(pkg)
	raw, err := db.Get(key)
	if err != nil {
		return err
	}
	if raw == nil {
		return errors.Wrapf(errors.ErrNotFound, "key %q", key)
	}
	if err := dst.Unmarshal(raw); err != nil {
		return errors.Wrapf(err, "unmarshal: key %q", key)
	}
	return nil
}

func dbkey(pkgName string) []byte {
	return []byte("_c:" + pkgName)
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
		return errors.Wrapf(errors.ErrNotFound, "no configuration in genesis for %q package", pkg)
	}
	if err := confOptions.ReadOptions(pkg, conf); err != nil {
		return errors.Wrapf(err, "read configuration for %s", pkg)
	}
	if err := Save(db, pkg, conf); err != nil {
		return errors.Wrapf(err, "save configuration for %s", pkg)
	}
	return nil
}

func NewConfigurationModelBucket() orm.ModelBucket {
	return &confModelBucket{}
}

// confModelBucket provides an access to configuration entities via
// orm.ModelBucket interface.
// This implementation is useful when used with x/lateinit package.
type confModelBucket struct{}

var _ orm.ModelBucket = (*confModelBucket)(nil)

func (c *confModelBucket) One(db weave.ReadOnlyKVStore, pkgName []byte, dest orm.Model) error {
	return Load(db, string(pkgName), dest)
}

func (c *confModelBucket) ByIndex(db weave.ReadOnlyKVStore, indexName string, key []byte, dest orm.ModelSlicePtr) (keys [][]byte, err error) {
	return nil, errors.Wrap(errors.ErrHuman, "not implemented")
}

func (c *confModelBucket) Put(db weave.KVStore, pkgName []byte, m orm.Model) ([]byte, error) {
	return pkgName, Save(db, string(pkgName), m)
}

func (c *confModelBucket) Delete(db weave.KVStore, pkgName []byte) error {
	return db.Delete(dbkey(string(pkgName)))
}

func (c *confModelBucket) Has(db weave.KVStore, pkgName []byte) error {
	ok, err := db.Has(dbkey(string(pkgName)))
	if err != nil {
		return err
	}
	if !ok {
		return errors.ErrNotFound
	}
	return nil
}

func (c *confModelBucket) Register(name string, r weave.QueryRouter) {
	panic("not implemented")
}
