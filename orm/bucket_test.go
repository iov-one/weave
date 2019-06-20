package orm

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestBucketName(t *testing.T) {
	obj := NewSimpleObj(nil, &Counter{})

	assert.Panics(t, func() {
		// An invalid bucket name must crash.
		NewBucket("l33t", obj)
	})
}

func TestBucketNameCollision(t *testing.T) {
	const bucketName = "mybucket"
	var objkey = []byte("collision-key")

	counter := &Counter{}
	assert.Nil(t, counter.Validate())
	o1 := NewSimpleObj(nil, counter)
	o1.SetKey([]byte(objkey))
	b1 := NewBucket(bucketName, o1)

	multiref := &MultiRef{
		Refs: [][]byte{
			[]byte("foobar"),
		},
	}
	assert.Nil(t, multiref.Validate())
	o2 := NewSimpleObj(nil, multiref)
	o2.SetKey([]byte(objkey))
	b2 := NewBucket(bucketName, o2)

	db := store.MemStore()
	assert.Nil(t, b1.Save(db, o1))

	// Buckets do not know about each other. Saving an object under the
	// same key overwrites and because there is no check of stored data,
	// this operation does not fail.
	if err := b2.Save(db, o2); err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	// Loading an object using the wrong bucket must fail because protobuf
	// deserialization cannot happen.
	if _, err := b1.Get(db, objkey); !errors.ErrState.Is(err) {
		t.Fatalf("cannot get object: %+v", err)
	}
}

func TestBucketCannotSaveInvalid(t *testing.T) {
	counter := &Counter{
		Count: -999, // Negative value is not valid.
	}
	if err := counter.Validate(); !errors.ErrState.Is(err) {
		t.Fatalf("unexpected error: %s", err)
	}

	o := NewSimpleObj(nil, counter)
	o.SetKey([]byte("mykey"))
	b := NewBucket("mybucket", o)

	db := store.MemStore()
	if err := b.Save(db, o); !errors.ErrState.Is(err) {
		t.Fatalf("invalid object must not save: %s", err)
	}
}

func TestBucketGetSave(t *testing.T) {
	counter := NewCounter(848)
	assert.Nil(t, counter.Validate())

	o := NewSimpleObj(nil, counter)
	o.SetKey([]byte("mykey"))
	b := NewBucket("mybucket", o)

	db := store.MemStore()
	if err := b.Save(db, o); err != nil {
		t.Fatalf("cannot save: %s", err)
	}

	res, err := b.Get(db, []byte("mykey"))
	if err != nil {
		t.Fatalf("cannot get object: %s", err)
	}

	c, ok := res.Value().(*Counter)
	if !ok {
		t.Fatalf("unexpected type: %s", err)
	}
	if c.Count != 848 {
		t.Fatalf("unexpected counter state: %d", c.Count)
	}

	// Update the counter state. This is a reference so the data
	// represented by `res` will be updated as well. Storing res in the
	// bucket must save the new state.
	c.Count = 59
	if err := b.Save(db, res); err != nil {
		t.Fatalf("cannot overwrite counter: %s", err)
	}

	res, err = b.Get(db, []byte("mykey"))
	if err != nil {
		t.Fatalf("cannot get overwritten object: %s", err)
	}
	if c, ok = res.Value().(*Counter); !ok {
		t.Fatalf("unexpected type: %s", err)
	} else if c.Count != 59 {
		t.Fatalf("unexpected counter state: %d", c.Count)
	}
}

// Make sure we have independent sequences.
func TestBucketSequence(t *testing.T) {
	b1 := NewBucket("aaa", NewSimpleObj(nil, &Counter{}))
	b2 := NewBucket("bbb", NewSimpleObj(nil, &Counter{}))

	db := store.MemStore()

	// Ensure they are sequential and not affecting one another. Repeat
	// this operation several times.
	for i := int64(1); i < 10; i++ {
		sa := b1.Sequence("seq1")
		a, err := sa.NextInt(db)
		assert.Nil(t, err)

		sb := b1.Sequence("seq2") // The same bucket but different name.
		b, err := sb.NextInt(db)
		assert.Nil(t, err)

		sc := b2.Sequence("seq1") // The same name but different bucket.
		c, err := sc.NextInt(db)
		assert.Nil(t, err)

		if a != i || a != b || a != c {
			t.Fatalf("different sequencces increment independently: a=%d b=%d c=%d", a, b, c)
		}
	}

}

// countByte is another index we can use
func countByte(obj Object) ([]byte, error) {
	if obj == nil {
		return nil, errors.Wrap(errors.ErrState, "Cannot take index of nil")
	}
	cntr, ok := obj.Value().(*Counter)
	if !ok {
		return nil, errors.Wrap(errors.ErrState, "Can only take index of Counter")
	}
	// last 8 bits...
	return bc(cntr.Count), nil
}

func bc(i int64) []byte {
	return []byte{byte(i % 256)}
}

func TestBucketSecondaryIndex(t *testing.T) {
	const uniq, mini = "uniq", "mini"

	bucket := NewBucket("special", NewSimpleObj(nil, new(Counter))).
		WithIndex(uniq, count, true).
		WithIndex(mini, countByte, false)

	a, b, c := []byte("a"), []byte("b"), []byte("c")
	oa := NewSimpleObj(a, NewCounter(5))
	oa2 := NewSimpleObj(a, NewCounter(245))
	ob := NewSimpleObj(b, NewCounter(256+5))
	ob2 := NewSimpleObj(b, NewCounter(245))
	oc := NewSimpleObj(c, NewCounter(512+245))

	type savecall struct {
		obj     Object
		wantErr *errors.Error
	}

	// query will query either by pattern or key
	// verifies that the proper results are returned
	type query struct {
		index   string
		like    Object
		at      []byte
		res     []Object
		wantErr *errors.Error
	}

	cases := map[string]struct {
		bucket  Bucket
		save    []savecall
		remove  [][]byte
		queries []query
	}{
		"insert one object enters into both indexes": {
			bucket: bucket,
			save:   []savecall{{obj: oa}},
			queries: []query{
				{uniq, oa, nil, []Object{oa}, nil},
				{mini, oa, nil, []Object{oa}, nil},
				{"foo", oa, nil, nil, ErrInvalidIndex},
			},
		},
		"add a second object and move one": {
			bucket: bucket,
			save: []savecall{
				{obj: oa},
				{obj: ob},
				{obj: oa2},
			},
			queries: []query{
				{uniq, oa, nil, nil, nil},
				{uniq, oa2, nil, []Object{oa2}, nil},
				{uniq, ob, nil, []Object{ob}, nil},
				{mini, nil, []byte{5}, []Object{ob}, nil},
				{mini, nil, []byte{245}, []Object{oa2}, nil},
			},
		},
		"prevent a conflicting save": {
			bucket: bucket,
			save: []savecall{
				{obj: oa2},
				{obj: ob2, wantErr: errors.ErrDuplicate},
			},
		},
		"update properly on delete as well": {
			bucket: bucket,
			save: []savecall{
				{obj: oa},
				{obj: ob2},
				{obj: oc},
			},
			remove: [][]byte{b},
			queries: []query{
				{uniq, oa, nil, []Object{oa}, nil},
				{uniq, ob2, nil, nil, nil},
				{uniq, oc, nil, []Object{oc}, nil},
				{mini, nil, []byte{5}, []Object{oa}, nil},
				{mini, nil, []byte{245}, []Object{oc}, nil},
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()

			for i, call := range tc.save {
				if err := tc.bucket.Save(db, call.obj); !call.wantErr.Is(err) {
					t.Fatalf("unexpected %d save call error: %s", i, err)
				}
			}

			for i, rem := range tc.remove {
				if err := tc.bucket.Delete(db, rem); err != nil {
					t.Fatalf("cannot remove %d: %s", i, err)
				}
			}

			for i, q := range tc.queries {
				var (
					res []Object
					err error
				)
				if q.like != nil {
					res, err = tc.bucket.GetIndexedLike(db, q.index, q.like)
				} else {
					res, err = tc.bucket.GetIndexed(db, q.index, q.at)
				}
				if !q.wantErr.Is(err) {
					t.Fatalf("unexpected %d query error: %s", i, err)
				}
				if !reflect.DeepEqual(q.res, res) {
					t.Fatalf("unexpected %d query result: %+v", i, res)
				}
			}
		})
	}
}

// Check query interface works, also with embedded indexes
func TestBucketQuery(t *testing.T) {
	// make some buckets for testing
	const mini = "mini"
	const uniq = "uniq"
	const name = "special"
	const bPath = "/special"
	const iPath = "/special/mini"
	const uiPath = "/special/uniq"

	// create a bucket with secondary index
	bucket := NewBucket("spec", NewSimpleObj(nil, new(Counter))).
		WithIndex(uniq, count, true).
		WithIndex(mini, countByte, false)

	// set up a router, using different name for bucket
	qr := weave.NewQueryRouter()
	bucket.Register(name, qr)

	// store some data here for init
	db := store.MemStore()
	a, b, c := []byte("aa"), []byte("aab"), []byte("caac")
	oa := NewSimpleObj(a, NewCounter(5))
	ob := NewSimpleObj(b, NewCounter(256+5))
	oc := NewSimpleObj(c, NewCounter(2))
	err := bucket.Save(db, oa)
	assert.Nil(t, err)
	err = bucket.Save(db, ob)
	assert.Nil(t, err)
	err = bucket.Save(db, oc)
	assert.Nil(t, err)

	toModel := func(t testing.TB, bucket Bucket, obj Object) weave.Model {
		t.Helper()

		dbkey := bucket.DBKey(obj.Key())
		val, err := obj.Value().Marshal()
		assert.Nil(t, err)
		return weave.Model{Key: dbkey, Value: val}
	}

	// these are the expected models with absolute keys
	// and serialized data
	dba := toModel(t, bucket, oa)
	dbb := toModel(t, bucket, ob)
	dbc := toModel(t, bucket, oc)
	assert.Equal(t, []byte("spec:aa"), dba.Key)
	if reflect.DeepEqual(dba.Value, dbb.Value) {
		t.Fatalf("various models data mixed up")
	}

	// these are query strings for index
	e5 := bc(5)
	e2 := bc(2)
	e77 := bc(77)

	cases := map[string]struct {
		path           string
		mod            string
		data           []byte
		missingHandler bool
		wantErr        *errors.Error
		expected       []weave.Model
	}{
		"bad path": {
			path:           bPath + "/",
			missingHandler: true,
		},
		"bad mod": {
			path:    bPath,
			mod:     "foo",
			data:    a,
			wantErr: errors.ErrInput,
		},
		"simple query - hit": {
			path:     bPath,
			data:     a,
			expected: []weave.Model{dba},
		},
		"simple query - miss": {
			path: bPath,
			data: []byte("a"),
		},
		"prefix query - multi hit": {
			path:     bPath,
			mod:      "prefix",
			data:     []byte("a"),
			expected: []weave.Model{dba, dbb},
		},
		"prefix query - miss": {
			path: bPath,
			mod:  "prefix",
			data: []byte("cc"),
		},
		"prefix query - all": {
			path:     bPath,
			mod:      "prefix",
			expected: []weave.Model{dba, dbb, dbc},
		},
		"simple index - miss": {
			path: iPath,
			data: e77,
		},
		"simple index - single hit": {
			path:     iPath,
			data:     e5,
			expected: []weave.Model{dba, dbb},
		},
		"simple index - multi": {
			path:     iPath,
			data:     e2,
			expected: []weave.Model{dbc},
		},
		"prefix index - miss": {
			path: iPath,
			mod:  "prefix",
			data: e77,
		},
		"prefix index - all (in order of index, last byte)": {
			path:     iPath,
			mod:      "prefix",
			expected: []weave.Model{dbc, dba, dbb},
		},
		"unique index - hit": {
			path:     uiPath,
			data:     encodeSequence(256 + 5),
			expected: []weave.Model{dbb},
		},
		"unique prefix index - all (in order of index, full count)": {
			path:     uiPath,
			mod:      "prefix",
			expected: []weave.Model{dbc, dba, dbb},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			qh := qr.Handler(tc.path)
			if tc.missingHandler {
				assert.Nil(t, qh)
				return
			}
			if qh == nil {
				t.Fatal("nil query handler")
			}

			res, err := qh.Query(db, tc.mod, tc.data)
			if !tc.wantErr.Is(err) {
				t.Fatalf("unexpected error: %s", err)
			}
			if err != nil {
				return
			}
			assert.Equal(t, tc.expected, res)
		})
	}
}

// Make sure saving indexes is a deterministic process. That is all writes
// happen in the same order.
func TestBucketIndexDeterministic(t *testing.T) {
	// Same as above, note there are two indexes. We can check the save
	// order.
	const uniq, mini = "uniq", "mini"
	bucket := NewBucket("special", NewSimpleObj(nil, new(Counter))).
		WithIndex(uniq, count, true).
		WithIndex(mini, countByte, false)

	key := []byte("key")
	val1 := NewSimpleObj(key, NewCounter(5))
	val2 := NewSimpleObj(key, NewCounter(256+5))

	db, log := store.LogableStore()

	ops := log.ShowOps()
	assert.Equal(t, 0, len(ops))

	// Saving one item should update the item as well as two indexes.
	assert.Nil(t, bucket.Save(db, val1))
	ops = log.ShowOps()
	assert.Equal(t, 3, len(ops))
	assertOps(t, ops, 3, 0)

	// saving second item should update the item as well as the one index that changed (don't write constant index)
	err := bucket.Save(db, val2)
	assert.Nil(t, err)
	ops = log.ShowOps()
	assert.Equal(t, 6, len(ops))
	assertOps(t, ops, 5, 1)

	// Now that we validated the "proper" ops, let's ensure all runs have
	// the same.
	for i := 0; i < 20; i++ {
		db2, log2 := store.LogableStore()
		assert.Nil(t, bucket.Save(db2, val1))
		assert.Nil(t, bucket.Save(db2, val2))
		ops2 := log2.ShowOps()
		assertSameOps(t, ops, ops2)
	}
}

func assertOps(t testing.TB, ops []store.Op, wantSet, wantDel int) {
	t.Helper()

	var set, del int
	for _, op := range ops {
		if op.IsSetOp() {
			set++
		} else {
			del++
		}
	}

	if set != wantSet {
		t.Errorf("want %d set operations, got %d", wantSet, set)
	}
	if del != wantDel {
		t.Errorf("want %d del operations, got %d", wantDel, del)
	}
}

func assertSameOps(t testing.TB, a, b []store.Op) {
	t.Helper()

	if len(a) != len(b) {
		t.Fatalf("different op count: %d != %d", len(a), len(b))
	}

	for i := range a {
		opa := a[i]
		opb := b[i]
		if opa.IsSetOp() != opb.IsSetOp() {
			t.Fatalf("set vs. delete difference at index %d", i)
		}
		if !bytes.Equal(opa.Key(), opb.Key()) {
			t.Fatalf("different key at index %d: %X vs %X", i, opa.Key(), opb.Key())
		}
	}
}
