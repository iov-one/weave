package migration

import (
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestSchemaVersionedBucket(t *testing.T) {
	const thisPkgName = "testpkg"

	reg := newRegister()

	reg.MustRegister(1, &MyModel{}, NoModification)
	reg.MustRegister(2, &MyModel{}, func(db weave.ReadOnlyKVStore, m Migratable) error {
		msg := m.(*MyModel)
		msg.Cnt += 2
		return msg.err
	})

	db := store.MemStore()

	ensureSchemaVersion(t, db, thisPkgName, 1)

	b := &MyModelBucket{
		Bucket: NewBucket(thisPkgName, "mymodel", orm.NewSimpleObj(nil, &MyModel{})),
	}

	// Use custom register instead of the global one to avoid pollution
	// from the application during tests.
	b.Bucket = b.Bucket.useRegister(reg)

	obj1 := orm.NewSimpleObj([]byte("schema_one"), &MyModel{
		Metadata: &weave.Metadata{Schema: 1},
		Cnt:      5,
	})
	assert.Nil(t, b.Save(db, obj1))

	if m, err := b.GetMyModel(db, "schema_one"); err != nil {
		t.Fatalf("cannot get model one: %s", err)
	} else if m.Metadata.Schema != 1 || m.Cnt != 5 {
		t.Fatalf("unexpected result model: %#v", m)
	}

	// Storing a model with a schema version higher than currently active
	// is not allowed.
	obj2 := orm.NewSimpleObj([]byte("schema_two"), &MyModel{
		Metadata: &weave.Metadata{Schema: 2},
		Cnt:      11,
	})
	if err := b.Save(db, obj2); !errors.ErrSchema.Is(err) {
		t.Fatalf("storing an object with an unknown schema version: %s", err)
	}

	// Bumping a schema should unlock saving entities with higher schema version.
	ensureSchemaVersion(t, db, thisPkgName, 2)

	if err := b.Save(db, obj2); err != nil {
		t.Fatalf("cannot save second object after schema version update: %s", err)
	}

	// Now that the schema was upgraded, all returned modlels must use it.
	// This means that returned models metadata schema must be set to two
	// and the payload must be updated.

	if m, err := b.GetMyModel(db, "schema_one"); err != nil {
		t.Fatalf("cannot get first model: %s", err)
	} else if m.Metadata.Schema != 2 || m.Cnt != 5+2 {
		t.Fatalf("unexpected result model: %#v", m)
	}

	if m, err := b.GetMyModel(db, "schema_two"); err != nil {
		t.Fatalf("cannot get second model: %s", err)
	} else if m.Metadata.Schema != 2 || m.Cnt != 11 {
		t.Fatalf("unexpected result model: %#v", m)
	}

	// Saving a model with an outdated schema must call the migration
	// before writing to the database.
	obj12 := orm.NewSimpleObj([]byte("schema_one_2"), &MyModel{
		Metadata: &weave.Metadata{Schema: 1},
		Cnt:      17,
	})
	assert.Nil(t, b.Save(db, obj12))
}

type MyModelBucket struct {
	Bucket
}

func (b *MyModelBucket) GetMyModel(db weave.KVStore, key string) (*MyModel, error) {
	obj, err := b.Get(db, []byte(key))
	if err != nil {
		return nil, err
	}
	if obj == nil || obj.Value() == nil {
		return nil, errors.Wrap(errors.ErrNotFound, "no such model")
	}
	m, ok := obj.Value().(*MyModel)
	if !ok {
		return nil, errors.Wrapf(errors.ErrModel, "invalid type: %T", obj.Value())
	}
	return m, nil
}

type MyModel struct {
	Metadata *weave.Metadata
	Cnt      int

	err error
}

func (m *MyModel) GetMetadata() *weave.Metadata {
	return m.Metadata
}

func (m *MyModel) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return err
	}
	return m.err
}

func (m *MyModel) Copy() orm.CloneableData {
	return &MyModel{
		Metadata: m.Metadata.Copy(),
		Cnt:      m.Cnt,
		err:      m.err,
	}
}

func (m *MyModel) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func (m *MyModel) Unmarshal(raw []byte) error {
	return json.Unmarshal(raw, &m)
}

var _ Migratable = (*MyModel)(nil)
var _ orm.CloneableData = (*MyModel)(nil)

func TestSchemaVersionedModelBucket(t *testing.T) {
	const thisPkgName = "testpkg"

	reg := newRegister()

	reg.MustRegister(1, &MyModel{}, NoModification)
	reg.MustRegister(2, &MyModel{}, func(db weave.ReadOnlyKVStore, m Migratable) error {
		msg := m.(*MyModel)
		msg.Cnt += 2
		return msg.err
	})

	db := store.MemStore()

	ensureSchemaVersion(t, db, thisPkgName, 1)

	b := NewModelBucket(
		thisPkgName,
		orm.NewModelBucket("mymodel", &MyModel{},
			orm.WithIndex("const", func(orm.Object) ([]byte, error) { return []byte("all"), nil }, false),
		),
	)

	// Use custom register instead of the global one to avoid pollution
	// from the application during tests.
	b.useRegister(reg)

	m1 := MyModel{
		Metadata: &weave.Metadata{Schema: 1},
		Cnt:      5,
	}
	k1, err := b.Put(db, nil, &m1)
	assert.Nil(t, err)

	var res MyModel
	if err := b.One(db, k1, &res); err != nil {
		t.Fatalf("cannot fetch the first model: %s", err)
	}
	assertMyModelState(t, &res, 1, 5)

	// Bumping a schema should unlock saving entities with higher schema version.
	ensureSchemaVersion(t, db, thisPkgName, 2)

	if err := b.One(db, k1, &res); err != nil {
		t.Fatalf("cannot fetch the first model: %s", err)
	}
	// Schema migration callback must update the model.
	assertMyModelState(t, &res, 2, 7)

	m2 := MyModel{
		Metadata: &weave.Metadata{Schema: 2},
		Cnt:      11,
	}
	k2, err := b.Put(db, nil, &m2)
	assert.Nil(t, err)
	if err := b.One(db, k2, &res); err != nil {
		t.Fatalf("cannot fetch the second model: %s", err)
	}
	// This model was stored with the second schema version so it must not
	// be updated.
	assertMyModelState(t, &res, 2, 11)

	// ByIndex must support destination being slice of values.
	var setp []*MyModel
	if _, err := b.ByIndex(db, "const", []byte("all"), &setp); err != nil {
		t.Fatalf("cannot query by index: %s", err)
	}
	wantp := []*MyModel{
		{Metadata: &weave.Metadata{Schema: 2}, Cnt: 7},
		{Metadata: &weave.Metadata{Schema: 2}, Cnt: 11},
	}
	assert.Equal(t, wantp, setp)

	// ByIndex must support destination being slice of pointers.
	var setv []MyModel
	if _, err := b.ByIndex(db, "const", []byte("all"), &setv); err != nil {
		t.Fatalf("cannot query by index: %s", err)
	}
	wantv := []MyModel{
		{Metadata: &weave.Metadata{Schema: 2}, Cnt: 7},
		{Metadata: &weave.Metadata{Schema: 2}, Cnt: 11},
	}
	assert.Equal(t, wantv, setv)

}

func assertMyModelState(t testing.TB, m *MyModel, wantSchemaVersion uint32, wantCnt int) {
	if m == nil {
		t.Fatal("MyModel instance is nil")
	}
	if m.Metadata == nil {
		t.Fatal("MyModel.Metadata is nil")
	}
	if m.Metadata.Schema != wantSchemaVersion {
		t.Fatalf("want schema version %d, got %d", wantSchemaVersion, m.Metadata.Schema)
	}
	if m.Cnt != wantCnt {
		t.Fatalf("want cnt %d, got %d", wantCnt, m.Cnt)
	}
}
