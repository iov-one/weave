package migration

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
)

type Bucket struct {
	orm.Bucket
	packageName string
	schema      *SchemaBucket
	migrations  *register
}

func NewBucket(packageName string, bucketName string, model orm.Cloneable) Bucket {
	return Bucket{
		Bucket:      orm.NewBucket(bucketName, model),
		packageName: packageName,
		schema:      NewSchemaBucket(),
		migrations:  reg,
	}
}

func (svb *Bucket) Get(db weave.ReadOnlyKVStore, key []byte) (orm.Object, error) {
	obj, err := svb.Bucket.Get(db, key)
	if err != nil || obj == nil {
		return obj, err
	}
	if err := svb.migrate(db, obj); err != nil {
		return obj, errors.Wrap(err, "migrate")
	}
	return obj, nil
}

func (svb *Bucket) Save(db weave.KVStore, obj orm.Object) error {
	if err := svb.migrate(db, obj); err != nil {
		return errors.Wrap(err, "migrate")
	}
	return svb.Bucket.Save(db, obj)
}

func (svb *Bucket) migrate(db weave.ReadOnlyKVStore, obj orm.Object) error {
	m, ok := obj.Value().(Migratable)
	if !ok {
		return errors.Wrap(errors.ErrInvalidModel, "model cannot be migrated")
	}
	currSchemaVer, err := svb.schema.CurrentSchema(db, svb.packageName)
	if err != nil {
		return errors.Wrap(err, "current model schema")
	}

	meta := m.GetMetadata()
	if meta == nil {
		return errors.Wrap(errors.ErrMetadata, "nil")
	}
	if meta.Schema > currSchemaVer {
		return errors.Wrapf(errors.ErrSchema, "model schema higher than %d", currSchemaVer)
	}

	// Migration is applied in place, directly modyfying the instance.
	if err := svb.migrations.Apply(db, m, currSchemaVer); err != nil {
		return errors.Wrap(err, "schema migration")
	}
	return nil
}
