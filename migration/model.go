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
	if err := s.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if s.Version < 1 {
		return errors.Wrap(errors.ErrModel, "version must be greater than zero")
	}
	if s.Pkg == "" {
		return errors.Wrap(errors.ErrModel, "pkg is required")
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

// schemaID returns a deterministic ID of this schema instance. Created IDs
// can be sorted using lexicographical order from the lowest to the highest
// version.
func schemaID(pkg string, version uint32) []byte {
	raw := make([]byte, len(pkg)+4)
	copy(raw, pkg)
	binary.BigEndian.PutUint32(raw[len(pkg):], version)
	return raw
}

type SchemaBucket struct {
	orm.Bucket
}

func NewSchemaBucket() *SchemaBucket {
	// Schema bucket is using plain orm.Bucket implementation so that is
	// can insert entities without schema version being registered. It
	// cannot use migration implementation bucket because it would cause
	// circular dependency on itself.
	b := orm.NewBucket("schema", orm.NewSimpleObj(nil, &Schema{}))
	return &SchemaBucket{Bucket: b}
}

// MustInitPkg initialize schema versioning for given package names. This
// registers a version one schema.
// This function panics if not successful. It is safe to call this function
// many times as duplicate registrations are ignored.
func MustInitPkg(db weave.KVStore, packageNames ...string) {
	for _, name := range packageNames {
		_, err := NewSchemaBucket().Create(db, &Schema{
			Metadata: &weave.Metadata{Schema: 1},
			Pkg:      name,
			Version:  1,
		})
		// Duplicated initializations are ignored.
		if err != nil && !errors.ErrDuplicate.Is(err) {
			panic(errors.Wrap(err, name))
		}
	}
}

// CurrentSchema returns the current version of the schema for a given package.
// It returns ErrNotFound if no schema version was registered for this package.
// Minimum schema version is 1.
func (b *SchemaBucket) CurrentSchema(db weave.ReadOnlyKVStore, packageName string) (uint32, error) {
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
			return 0, errors.Wrap(errors.ErrNotFound, "not initialized")
		}
		return ver - 1, nil
	}
	return 0, errors.Wrap(errors.ErrState, "version too high")
}

func (b *SchemaBucket) Get(db weave.KVStore, key []byte) error {
	// Prevent direct access to the bucket content. Use CurrentSchema method instead.
	return errors.Wrap(errors.ErrHuman, "this bucket does not allow for a direct value access")
}

// Save persists the state of a given schema entity.
func (b *SchemaBucket) Save(db weave.KVStore, obj orm.Object) error {
	s, ok := obj.Value().(*Schema)
	if !ok {
		return errors.Wrapf(errors.ErrModel, "invalid type: %T", obj.Value())
	}
	if err := b.validateNextSchema(db, s); err != nil {
		return err
	}
	return b.Bucket.Save(db, obj)
}

// Create adds given schema instance to the store and returns the ID of the
// newly inserted entity.
func (b *SchemaBucket) Create(db weave.KVStore, s *Schema) (orm.Object, error) {
	if err := b.validateNextSchema(db, s); err != nil {
		return nil, err
	}
	obj := orm.NewSimpleObj(schemaID(s.Pkg, s.Version), s)
	return obj, b.Bucket.Save(db, obj)
}

// validateNextSchema returns an error if given Schema instance is does not
// represent the next valid schema version.
func (b *SchemaBucket) validateNextSchema(db weave.KVStore, next *Schema) error {
	ver, err := b.CurrentSchema(db, next.Pkg)
	if err != nil {
		if errors.ErrNotFound.Is(err) {
			ver = 0
			if next.Version != 1 {
				return errors.Wrap(errors.ErrInput, "schema not initialized with version 1")
			}
		} else {
			return errors.Wrap(err, "current schema")
		}
	}
	if ver+1 != next.Version {
		// Schema versioning is sequential and the numbers must be incrementing.
		return errors.Wrapf(errors.ErrDuplicate, "previous schema is %d", ver)
	}
	return nil
}

// RegisterQuery registers schema bucket for querying.
func RegisterQuery(qr weave.QueryRouter) {
	NewSchemaBucket().Register("schemas", qr)
}
