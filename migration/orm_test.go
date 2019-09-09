package migration

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
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
		Bucket: NewBucket(thisPkgName, "mymodel", &MyModel{}),
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

type MySerialModel struct {
	Metadata *weave.Metadata
	ID       []byte
	Cnt      int

	err error
}

func (m *MySerialModel) GetMetadata() *weave.Metadata {
	return m.Metadata
}

func (m *MySerialModel) SetID(id []byte) error {
	m.ID = id
	return nil
}

func (m *MySerialModel) GetID() []byte {
	return m.ID
}

func (m *MySerialModel) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return err
	}
	return m.err
}

func (m *MySerialModel) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func (m *MySerialModel) Unmarshal(raw []byte) error {
	return json.Unmarshal(raw, &m)
}

var _ Migratable = (*MySerialModel)(nil)
var _ orm.CloneableData = (*MySerialModel)(nil)

type MySerialModelWithRef struct {
	Metadata        *weave.Metadata
	ID              []byte
	MySerialModelID []byte
	Cnt             int

	err error
}

func (m *MySerialModelWithRef) GetMetadata() *weave.Metadata {
	return m.Metadata
}

func (m *MySerialModelWithRef) SetID(id []byte) error {
	m.ID = id
	return nil
}

func (m *MySerialModelWithRef) GetID() []byte {
	return m.ID
}

func (m *MySerialModelWithRef) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return err
	}
	return m.err
}

func (m *MySerialModelWithRef) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func (m *MySerialModelWithRef) Unmarshal(raw []byte) error {
	return json.Unmarshal(raw, &m)
}

var _ Migratable = (*MySerialModelWithRef)(nil)
var _ orm.CloneableData = (*MySerialModelWithRef)(nil)

func TestSchemaVersionedSerialModelBucket(t *testing.T) {
	const thisPkgName = "testpkg"

	reg := newRegister()

	reg.MustRegister(1, &MySerialModel{}, NoModification)
	reg.MustRegister(2, &MySerialModel{}, func(db weave.ReadOnlyKVStore, m Migratable) error {
		msg := m.(*MySerialModel)
		msg.Cnt += 2
		return msg.err
	})

	db := store.MemStore()

	ensureSchemaVersion(t, db, thisPkgName, 1)

	b1 := NewSerialModelBucket(
		thisPkgName,
		orm.NewSerialModelBucket("mysmodel", &MySerialModel{},
			orm.WithIndexSerial("const", func(orm.Object) ([]byte, error) { return []byte("all"), nil }, false),
		),
	)

	// Use custom register instead of the global one to avoid pollution
	// from the application during tests.
	b1.useRegister(reg)

	m1ID := weavetest.SequenceID(1)
	m1 := MySerialModel{
		Metadata: &weave.Metadata{Schema: 1},
		ID:       m1ID,
		Cnt:      1,
	}
	err := b1.Put(db, &m1)
	assert.Nil(t, err)

	var res MySerialModel
	if err := b1.One(db, m1.ID, &res); err != nil {
		t.Fatalf("cannot fetch the first model: %s", err)
	}
	assertMySerialModelState(t, &res, 1, 1)

	// Bumping a schema should unlock saving entities with higher schema version.
	ensureSchemaVersion(t, db, thisPkgName, 2)

	if err := b1.One(db, m1.ID, &res); err != nil {
		t.Fatalf("cannot fetch the first model: %s", err)
	}
	// Schema migration callback must update the model.
	assertMySerialModelState(t, &res, 2, 3)

	m2ID := weavetest.SequenceID(2)
	m2 := MySerialModel{
		Metadata: &weave.Metadata{Schema: 2},
		ID:       m2ID,
		Cnt:      11,
	}
	err = b1.Put(db, &m2)
	assert.Nil(t, err)
	if err := b1.One(db, m2.ID, &res); err != nil {
		t.Fatalf("cannot fetch the second model: %s", err)
	}
	// This model was stored with the second schema version so it must not
	// be updated.
	assertMySerialModelState(t, &res, 2, 11)

	// ByIndex must support destination being slice of pointers.
	var setv []MySerialModel
	if err := b1.ByIndex(db, "const", []byte("all"), &setv); err != nil {
		t.Fatalf("cannot query by index: %s", err)
	}
	wantv := []MySerialModel{
		{Metadata: &weave.Metadata{Schema: 2}, ID: m1ID, Cnt: 3},
		{Metadata: &weave.Metadata{Schema: 2}, ID: m2ID, Cnt: 11},
	}
	assert.Equal(t, wantv, setv)

}

// TestSchemaVersionedSerialModelBucketRefID tests if external ID
// modified during migration
func TestSchemaVersionedSerialModelBucketRefID(t *testing.T) {
	const thisPkgName = "testpkg"

	reg := newRegister()

	// Register MySerialModel versions
	reg.MustRegister(1, &MySerialModel{}, NoModification)
	reg.MustRegister(2, &MySerialModel{}, func(db weave.ReadOnlyKVStore, m Migratable) error {
		msg := m.(*MySerialModel)
		msg.Cnt++
		return msg.err
	})

	// Register MySerialModelWithRef versions
	reg.MustRegister(1, &MySerialModelWithRef{}, NoModification)
	reg.MustRegister(2, &MySerialModelWithRef{}, func(db weave.ReadOnlyKVStore, m Migratable) error {
		// Never manipulate ID
		msg := m.(*MySerialModelWithRef)
		msg.Cnt += 2
		return msg.err
	})

	// Initilize MySerialModel bucket
	b1 := NewSerialModelBucket(
		thisPkgName,
		orm.NewSerialModelBucket("mysmodel", &MySerialModel{},
			orm.WithIndexSerial("const", func(orm.Object) ([]byte, error) { return []byte("all"), nil }, false),
		),
	)

	// Initilize MySerialModelWithRef bucket
	b2 := NewSerialModelBucket(
		thisPkgName,
		orm.NewSerialModelBucket("mysmodelr", &MySerialModelWithRef{},
			orm.WithIndexSerial("const", func(orm.Object) ([]byte, error) { return []byte("all"), nil }, false),
		),
	)

	// Use custom register instead of the global one to avoid pollution
	// from the application during tests.
	b1.useRegister(reg)
	b2.useRegister(reg)

	db := store.MemStore()

	ensureSchemaVersion(t, db, thisPkgName, 1)

	// Iniatilize MySerialModel and save
	mID := weavetest.SequenceID(1)
	m := MySerialModel{
		Metadata: &weave.Metadata{Schema: 1},
		ID:       mID,
		Cnt:      1,
	}
	err := b1.Put(db, &m)
	assert.Nil(t, err)

	var mres MySerialModel
	if err := b1.One(db, m.ID, &mres); err != nil {
		t.Fatalf("cannot fetch the model: %s", err)
	}

	assertMySerialModelState(t, &mres, 1, 1)

	// Initilize MySerialModel with external reference
	mwrID := weavetest.SequenceID(1)
	mwr := MySerialModelWithRef{
		Metadata:        &weave.Metadata{Schema: 1},
		ID:              mwrID,
		MySerialModelID: mID,
		Cnt:             2,
	}

	// Save MySerialModel with external reference
	err = b2.Put(db, &mwr)
	assert.Nil(t, err)

	var mwres MySerialModelWithRef
	if err := b2.One(db, mwr.ID, &mwres); err != nil {
		t.Fatalf("cannot fetch the model: %s", err)
	}

	assertMySerialModelWithRefState(t, &mwres, 1, 2)

	// Bump schema version
	ensureSchemaVersion(t, db, thisPkgName, 2)

	if err := b1.One(db, m.ID, &mres); err != nil {
		t.Fatalf("cannot fetch the model: %s", err)
	}
	// Check if migration applied successfully
	assertMySerialModelState(t, &mres, 2, 2)

	if err := b2.One(db, mwr.ID, &mwres); err != nil {
		t.Fatalf("cannot fetch the model with reference: %s", err)
	}
	// Check if migration applied successfully
	assertMySerialModelWithRefState(t, &mwres, 2, 4)

	// Check if m1 ID and MySerialModelWithRefs external ID matches
	if bytes.Compare(mwres.MySerialModelID, mres.ID) != 0 {
		t.Fatalf("id %d does not match reference id %d", mwres.MySerialModelID, mres.ID)
	}

	// Check if MySerialModel is accessible with using ID retrieved
	// from MySerialModelWithRef
	if err := b1.One(db, mwres.MySerialModelID, &mres); err != nil {
		t.Fatalf("cannot fetch the model with reference: %s", err)
	}
}

func TestSchemaVersionedSerialModelBucketPrefixScan(t *testing.T) {
	// Initialize register
	const thisPkgName = "testpkg"

	reg := newRegister()

	// Register MySerialModel versions
	reg.MustRegister(1, &MySerialModel{}, NoModification)
	reg.MustRegister(2, &MySerialModel{}, func(db weave.ReadOnlyKVStore, m Migratable) error {
		msg := m.(*MySerialModel)
		msg.Cnt++
		return msg.err
	})

	// Initialize bucket
	b := NewSerialModelBucket(
		thisPkgName,
		orm.NewSerialModelBucket("mysmodel", &MySerialModel{}),
	)

	// Initialize db
	db := store.MemStore()

	// Initialize schema
	ensureSchemaVersion(t, db, thisPkgName, 1)

	// Initalize models
	m1ID := weavetest.SequenceID(1)
	m2ID := weavetest.SequenceID(2)
	m3ID := weavetest.SequenceID(3)
	models := []MySerialModel{
		MySerialModel{
			Metadata: &weave.Metadata{Schema: 1},
			ID:       m1ID,
			Cnt:      1,
		},
		MySerialModel{
			Metadata: &weave.Metadata{Schema: 1},
			ID:       m2ID,
			Cnt:      5,
		},
		MySerialModel{
			Metadata: &weave.Metadata{Schema: 1},
			ID:       m3ID,
			Cnt:      10,
		},
		MySerialModel{
			Metadata: &weave.Metadata{Schema: 1},
			// ID must be Sequence(4)
			Cnt: 15,
		},
	}

	// Save models
	for i := range models {
		err := b.Put(db, &models[i])
		assert.Nil(t, err)
	}

	// migrate
	ensureSchemaVersion(t, db, thisPkgName, 2)

	// check
	iter, err := b.PrefixScan(db, []byte{0}, false)
	defer iter.Release()
	assert.Nil(t, err)

	var m MySerialModel
	iter.LoadNext(&m)
	assertMySerialModelState(t, &m, 2, 2)
	iter.LoadNext(&m)
	assertMySerialModelState(t, &m, 2, 6)
	iter.LoadNext(&m)
	assertMySerialModelState(t, &m, 2, 11)
	iter.LoadNext(&m)
	assertMySerialModelState(t, &m, 2, 16)
}

func assertMySerialModelState(t testing.TB, m *MySerialModel, wantSchemaVersion uint32, wantCnt int) {
	if m == nil {
		t.Fatal("MySerialModel instance is nil")
	}
	if m.Metadata == nil {
		t.Fatal("MySerialModel.Metadata is nil")
	}
	if m.Metadata.Schema != wantSchemaVersion {
		t.Fatalf("want schema version %d, got %d", wantSchemaVersion, m.Metadata.Schema)
	}
	if err := orm.ValidateSequence(m.ID); err != nil {
		t.Fatalf("id %d is not a sequence", m.ID)
	}
	if m.Cnt != wantCnt {
		t.Fatalf("want cnt %d, got %d", wantCnt, m.Cnt)
	}
}

func assertMySerialModelWithRefState(t testing.TB, m *MySerialModelWithRef, wantSchemaVersion uint32, wantCnt int) {
	if m == nil {
		t.Fatal("MySerialModelWithRef instance is nil")
	}
	if m.Metadata == nil {
		t.Fatal("MySerialModelWithRef.Metadata is nil")
	}
	if m.Metadata.Schema != wantSchemaVersion {
		t.Fatalf("want schema version %d, got %d", wantSchemaVersion, m.Metadata.Schema)
	}
	if err := orm.ValidateSequence(m.ID); err != nil {
		t.Fatalf("id %d is not a sequence", m.ID)
	}
	if err := orm.ValidateSequence(m.MySerialModelID); err != nil {
		t.Fatalf("extenal id %d is not a sequence", m.MySerialModelID)
	}
	if m.Cnt != wantCnt {
		t.Fatalf("want cnt %d, got %d", wantCnt, m.Cnt)
	}
}
