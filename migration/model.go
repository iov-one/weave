package migration

import (
	"encoding/binary"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
)

var _ orm.CloneableData = (*Schema)(nil)

func init() {
	MustRegister(1, &Schema{}, NoModification)
}

func (s *Schema) Validate() error {
	if s.Version < 1 {
		return errors.Wrap(errors.ErrInvalidModel, "version must be greater than zero")
	}
	if s.Pkg == "" {
		return errors.Wrap(errors.ErrInvalidModel, "pkg is required")
	}
	return nil
}

func (s *Schema) Copy() orm.CloneableData {
	return &Schema{
		Metadata: s.Metadata.Copy(),
		Version:  s.Version,
		Pkg:      s.Pkg,
	}
}

// schemaID returns a detereministic ID of this schema instance.
func schemaID(pkg string, version uint32) []byte {
	raw := make([]byte, len(pkg)+4)
	copy(raw, pkg)
	binary.LittleEndian.PutUint32(raw[len(pkg):], version)
	return raw
}

type SchemaBucket struct {
	orm.Bucket
}

func NewSchemaBucket() *SchemaBucket {
	b := orm.NewBucket("schema", orm.NewSimpleObj(nil, &Schema{}))
	return &SchemaBucket{Bucket: b}
}

// CurrentSchema returns the current version of the schema for a given package.
// It returns ErrNotFound if no schema version was registered for this package.
// Minimum schema version is 1.
func (b *SchemaBucket) CurrentSchema(db weave.KVStore, packageName string) (uint32, error) {
	for ver := uint32(1); ver < 10000; ver++ {
		key := schemaID(packageName, ver)
		obj, err := b.Bucket.Get(db, key)
		if err != nil {
			return 0, errors.Wrap(err, "bucket get")
		}
		if obj != nil {
			continue
		}
		if ver == 1 {
			return 0, errors.Wrap(errors.ErrNotFound, "not registered")
		}
		return ver - 1, nil
	}
	return 0, errors.Wrap(errors.ErrInvalidState, "version too high")
}

// Save persists the state of a given schema entity.
func (b *SchemaBucket) Save(db weave.KVStore, obj orm.Object) error {
	s, ok := obj.Value().(*Schema)
	if !ok {
		return errors.Wrapf(errors.ErrInvalidModel, "invalid type: %T", obj.Value())
	}
	switch ver, err := b.CurrentSchema(db, s.Pkg); {
	case err != nil:
		return errors.Wrap(err, "current schema")
	case ver+1 != s.Version:
		// Schema versioning is sequential and the numbers must be incrementing.
		return errors.Wrapf(errors.ErrInvalidInput, "previous schema is %d", ver)
	}
	return b.Bucket.Save(db, obj)
}

// Create adds given schema instance to the store and returns the ID of the
// newly inserted entity.
func (b *SchemaBucket) Create(db weave.KVStore, s *Schema) (orm.Object, error) {
	switch ver, err := b.CurrentSchema(db, s.Pkg); {
	case err != nil:
		return nil, errors.Wrap(err, "current schema")
	case ver+1 != s.Version:
		// Schema versioning is sequential and the numbers must be incrementing.
		return nil, errors.Wrapf(errors.ErrInvalidInput, "previous schema is %d", ver)
	}
	obj := orm.NewSimpleObj(schemaID(s.Pkg, s.Version), s)
	return obj, b.Bucket.Save(db, obj)
}
