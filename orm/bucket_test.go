package orm

import (
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type saver func(Object) error
type transformer func(Object, saver) error

func set(key []byte, n int64) transformer {
	return func(obj Object, save saver) error {
		if obj != nil {
			return errors.New("expected empty")
		}
		obj = NewSimpleObj(key, NewCounter(n))
		return save(obj)
	}
}

func addN(expect, n int64) transformer {
	return func(obj Object, save saver) error {
		if obj == nil {
			return errors.New("expected non-nil value")
		}
		cntr, ok := obj.Value().(*Counter)
		if !ok {
			return errors.New("expected counter")
		}
		if cntr.Count != expect {
			return errors.Errorf("Expected %d, got %d", expect, cntr.Count)
		}
		cntr.Count += n
		return save(obj)
	}
}

func isEmpty(obj Object, save saver) error {
	if obj != nil {
		return errors.New("Expected empty object")
	}
	return nil
}

// Test get/save on one bucket
// Test get/save are independent between buckets
// Test bucket names enforced
func TestBucketStore(t *testing.T) {
	// make some buckets for testing
	counter := NewSimpleObj(nil, new(Counter))
	multi := NewSimpleObj(nil, new(MultiRef))

	count := NewBucket("some", counter)
	count2 := NewBucket("somet", counter)
	bad := NewBucket("some", multi)
	assert.Panics(t, func() { NewBucket("l33t", counter) })

	// default key to check for conflicts with names
	k := []byte{'t', ':', 'b'}
	k2 := []byte{'b'}

	cases := []struct {
		bucket    Bucket
		get       []byte
		transform transformer
		isError   bool
	}{
		0: {count, k, isEmpty, false},
		1: {count, k, set(k, 55), false},
		2: {count, k, isEmpty, true},
		3: {count, k, addN(55, 22), false},
		// this reads wrong type, so makes error
		4: {bad, k, nil, true},
		5: {bad, k2, isEmpty, false},
		// add more values and check no overlap
		6:  {count, k2, set(k, 17), false},
		7:  {count2, k, isEmpty, false},
		8:  {count2, k2, isEmpty, false},
		9:  {count2, k2, set(k2, 99), false},
		10: {count2, k2, addN(99, 1), false},
		11: {count2, k, isEmpty, false},
		12: {count2, k2, isEmpty, true},
		// make sure negaitves cannot be stored
		13: {count, k2, addN(17, -20), true},
	}

	db := store.MemStore()
	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			b := tc.bucket
			s := func(o Object) error { return b.Save(db, o) }

			var obj Object
			var err error
			if tc.get != nil {
				obj, err = b.Get(db, tc.get)
				if err != nil {
					require.True(t, tc.isError, "%v", err)
					return
				}
			}

			if tc.transform != nil {
				err = tc.transform(obj, s)
				if tc.isError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			}
		})
	}
}

// make sure we have independent sequences
func TestBucketSequence(t *testing.T) {
	// make some buckets for testing
	counter := NewSimpleObj(nil, new(Counter))
	a := NewBucket("many", counter)
	b := NewBucket("man", counter)

	s1 := "ard"
	s2 := "yard"
	cases := []struct {
		bucket Bucket
		seq    string
		add    int64
		expect int64
	}{
		// check the two sequences are both saved and independent
		{a, s1, 5, 5},
		{a, s1, 6, 11},
		{a, s2, 7, 7},
		{a, s2, 12, 19},
		{a, s1, 6, 17},
		// check there is no interplay between the two buckets
		{b, s1, 22, 22},
		{b, s2, 99, 99},
		{b, s1, 118, 140},
	}

	db := store.MemStore()
	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			s := tc.bucket.Sequence(tc.seq)
			res, _ := s.increment(db, tc.add)
			assert.Equal(t, tc.expect, res)
		})
	}

}

// countByte is another index we can use
func countByte(obj Object) ([]byte, error) {
	if obj == nil {
		return nil, errors.New("Cannot take index of nil")
	}
	cntr, ok := obj.Value().(*Counter)
	if !ok {
		return nil, errors.New("Can only take index of Counter")
	}
	// last 8 bits...
	return bc(cntr.Count), nil
}

func bc(i int64) []byte {
	return []byte{byte(i % 256)}
}

// query will query either by pattern or key
// verifies that the proper results are returned
type query struct {
	index   string
	like    Object
	at      []byte
	res     []Object
	isError bool
}

func (q query) check(t *testing.T, b Bucket, db weave.KVStore) {
	var res []Object
	var err error
	if q.like != nil {
		res, err = b.GetIndexedLike(db, q.index, q.like)
	} else {
		res, err = b.GetIndexed(db, q.index, q.at)
	}
	if q.isError {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
		assert.EqualValues(t, q.res, res)
	}
}

// Make sure secondary indexes work
func TestBucketIndex(t *testing.T) {
	// make some buckets for testing
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

	cases := []struct {
		bucket    Bucket
		save      []Object
		saveError bool
		remove    [][]byte
		queries   []query
	}{
		// insert one object enters into both indexes
		0: {
			bucket, []Object{oa}, false, nil,
			[]query{
				{uniq, oa, nil, []Object{oa}, false},
				{mini, oa, nil, []Object{oa}, false},
				{"foo", oa, nil, nil, true},
			},
		},
		// add a second object and move one
		1: {
			bucket, []Object{oa, ob, oa2}, false, nil,
			[]query{
				{uniq, oa, nil, nil, false},
				{uniq, oa2, nil, []Object{oa2}, false},
				{uniq, ob, nil, []Object{ob}, false},
				{mini, nil, []byte{5}, []Object{ob}, false},
				{mini, nil, []byte{245}, []Object{oa2}, false},
			},
		},
		// prevent a conflicting save
		2: {
			bucket, []Object{oa2, ob2}, true, nil, nil,
		},
		// update properly on delete as well
		3: {
			bucket, []Object{oa, ob2, oc}, false, [][]byte{b},
			[]query{
				{uniq, oa, nil, []Object{oa}, false},
				{uniq, ob2, nil, nil, false},
				{uniq, oc, nil, []Object{oc}, false},
				{mini, nil, []byte{5}, []Object{oa}, false},
				{mini, nil, []byte{245}, []Object{oc}, false},
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			db := store.MemStore()
			b := tc.bucket

			// add all initial objects and enforce
			// error or no error
			hasErr := false
			for _, s := range tc.save {
				err := b.Save(db, s)
				if !tc.saveError {
					require.NoError(t, err)
				} else if err != nil {
					hasErr = true
				}
			}
			if tc.saveError {
				require.True(t, hasErr)
				return
			}

			// remove any if desired
			for _, rem := range tc.remove {
				err := b.Delete(db, rem)
				require.NoError(t, err)
			}

			for _, q := range tc.queries {
				q.check(t, b, db)
			}
		})
	}
}

func toModel(t *testing.T, bucket Bucket, obj Object) weave.Model {
	dbkey := bucket.DBKey(obj.Key())
	val, err := obj.Value().Marshal()
	require.NoError(t, err)
	return weave.Model{Key: dbkey, Value: val}
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
	require.NoError(t, err)
	err = bucket.Save(db, ob)
	require.NoError(t, err)
	err = bucket.Save(db, oc)
	require.NoError(t, err)

	// these are the expected models with absolute keys
	// and serialized data
	dba := toModel(t, bucket, oa)
	dbb := toModel(t, bucket, ob)
	dbc := toModel(t, bucket, oc)
	require.Equal(t, []byte("spec:aa"), dba.Key)
	require.NotEqual(t, dba.Value, dbb.Value)

	// these are query strings for index
	e5 := bc(5)
	e2 := bc(2)
	e77 := bc(77)

	cases := []struct {
		path           string
		mod            string
		data           []byte
		missingHandler bool
		isError        bool
		expected       []weave.Model
	}{
		// bad path
		0: {
			bPath + "/", "", a, true, true, nil,
		},
		// bad mod
		1: {
			bPath, "foo", a, false, true, nil,
		},
		// simple query - hit
		2: {
			bPath, "", a, false, false, []weave.Model{dba},
		},
		// simple query - miss
		3: {
			bPath, "", []byte("a"), false, false, nil,
		},
		// prefix query - multi hit
		4: {
			bPath, "prefix", []byte("a"), false, false,
			[]weave.Model{dba, dbb},
		},
		// prefix query - miss
		5: {
			bPath, "prefix", []byte("cc"), false, false, nil,
		},
		// prefix query - all
		6: {
			bPath, "prefix", nil, false, false,
			[]weave.Model{dba, dbb, dbc},
		},
		// simple index - miss
		7: {
			iPath, "", e77, false, false, nil,
		},
		// simple index - single hit
		8: {
			iPath, "", e5, false, false, []weave.Model{dba, dbb},
		},
		// simple index - multi
		9: {
			iPath, "", e2, false, false, []weave.Model{dbc},
		},
		// prefix index - miss
		10: {
			iPath, "prefix", e77, false, false, nil,
		},
		// prefix index - all (in order of index, last byte)
		11: {
			iPath, "prefix", nil, false, false, []weave.Model{dbc, dba, dbb},
		},
		// unique index - hit
		12: {
			uiPath, "", encodeSequence(256 + 5), false, false, []weave.Model{dbb},
		},
		// unique prefix index - all (in order of index, full count)
		13: {
			uiPath, "prefix", nil, false, false, []weave.Model{dbc, dba, dbb},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			qh := qr.Handler(tc.path)
			if tc.missingHandler {
				require.Nil(t, qh)
				return
			}
			require.NotNil(t, qh)

			res, err := qh.Query(db, tc.mod, tc.data)
			if tc.isError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.EqualValues(t, tc.expected, res)
		})
	}
}
