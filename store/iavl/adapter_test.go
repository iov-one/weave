package iavl

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/confio/weave/store"
)

type Model = store.Model
type Op = store.Op

// makeBase returns the base layer
//
// If you want to test a different kvstore implementation
// you can copy most of these tests and change makeBase.
// Once that passes, customize and extend as you wish
func makeBase() store.CacheableKVStore {
	commit := NewCommitStore("/tmp")
	commit.LoadLatestVersion()
	return commit.Adapter()
}

// TestBTreeCacheGetSet does basic sanity checks on our cache
//
// Other tests should handle deletes, setting same value,
// iterating over ranges, and general fuzzing
func TestBTreeCacheGetSet(t *testing.T) {
	base := makeBase()

	// make sure the btree is empty at start but returns results
	// that are writen to it
	k, v := []byte("french"), []byte("fry")
	assert.Nil(t, base.Get(k))
	assert.False(t, base.Has(k))
	base.Set(k, v)
	assert.Equal(t, v, base.Get(k))
	assert.True(t, base.Has(k))

	// now layer another btree on top and make sure that we get
	// base data
	cache := base.CacheWrap()
	assert.Equal(t, v, cache.Get(k))
	assert.True(t, cache.Has(k))

	// writing more data is only visible in the cache
	k2, v2 := []byte("LA"), []byte("Dodgers")
	assert.Nil(t, cache.Get(k2))
	assert.False(t, cache.Has(k2))
	cache.Set(k2, v2)
	assert.Equal(t, v2, cache.Get(k2))
	assert.Nil(t, base.Get(k2))
	assert.True(t, cache.Has(k2))
	assert.False(t, base.Has(k2))

	// we can write the cache to the base layer...
	cache.Write()
	assert.Equal(t, v, base.Get(k))
	assert.Equal(t, v2, base.Get(k2))
	assert.True(t, base.Has(k))
	assert.True(t, base.Has(k2))

	// we can discard one
	k3, v3 := []byte("Bayern"), []byte("Munich")
	c2 := base.CacheWrap()
	assert.Equal(t, v, c2.Get(k))
	assert.Equal(t, v2, c2.Get(k2))
	c2.Set(k3, v3)
	c2.Discard()

	// and commit another
	c3 := base.CacheWrap()
	assert.Equal(t, v, c3.Get(k))
	assert.Equal(t, v2, c3.Get(k2))
	c3.Delete(k)
	c3.Write()

	// make sure it commits proper
	assert.Nil(t, base.Get(k))
	assert.Equal(t, v2, base.Get(k2))
	assert.Nil(t, base.Get(k3))
}

// TestBTreeCacheConflicts checks that we can handle
// overwriting values and deleting underlying values
func TestBTreeCacheConflicts(t *testing.T) {
	// make 10 keys and 20 values....
	ks := randKeys(10, 16)
	vs := randKeys(20, 40)

	cases := [...]struct {
		parentOps     []Op
		childOps      []Op
		parentQueries []Model // Key is what we query, Value is what we espect
		childQueries  []Model // Key is what we query, Value is what we espect
	}{
		// overwrite one, delete another, add a third
		0: {
			[]Op{store.SetOp(ks[1], vs[1]), store.SetOp(ks[2], vs[2])},
			[]Op{store.SetOp(ks[1], vs[11]), store.SetOp(ks[3], vs[7]), store.DelOp(ks[2])},
			[]Model{store.Pair(ks[1], vs[1]), store.Pair(ks[2], vs[2]), store.Pair(ks[3], nil)},
			[]Model{store.Pair(ks[1], vs[11]), store.Pair(ks[2], nil), store.Pair(ks[3], vs[7])},
		},
	}

	for i, tc := range cases {
		parent := makeBase()
		for _, op := range tc.parentOps {
			op.Apply(parent)
		}

		child := parent.CacheWrap()
		for _, op := range tc.childOps {
			op.Apply(child)
		}

		// now check the parent is unaffected
		for j, q := range tc.parentQueries {
			res := parent.Get(q.Key)
			assert.Equal(t, q.Value, res, "%d / %d", i, j)
			has := parent.Has(q.Key)
			assert.Equal(t, q.Value != nil, has, "%d / %d", i, j)
		}

		// the child shows changes
		for j, q := range tc.childQueries {
			res := child.Get(q.Key)
			assert.Equal(t, q.Value, res, "%d / %d", i, j)
			has := child.Has(q.Key)
			assert.Equal(t, q.Value != nil, has, "%d / %d", i, j)
		}

		// write child to parent and make sure it also shows proper data
		child.Write()
		for j, q := range tc.childQueries {
			res := parent.Get(q.Key)
			assert.Equal(t, q.Value, res, "%d / %d", i, j)
			has := parent.Has(q.Key)
			assert.Equal(t, q.Value != nil, has, "%d / %d", i, j)
		}
	}
}

// TestFuzzBTreeCacheIterator makes sure the basic iterator
// works. Includes random deletes, but not nested iterators.
func TestFuzzBTreeCacheIterator(t *testing.T) {
	const Size = 50
	const DeleteCount = 20

	toSet := randModels(Size, 8, 40)
	toDel := randModels(DeleteCount, 8, 40)
	expect := sortModels(toSet)
	ops := append(
		makeSetOps(toSet...),
		makeDelOps(toDel...)...)

	parentSet := randModels(Size, 8, 40)
	parentDel := randModels(DeleteCount, 8, 40)
	parentOps := append(
		makeSetOps(parentSet...),
		makeDelOps(parentDel...)...)

	both := sortModels(append(toSet, parentSet...))

	cases := [...]iterCase{
		// just write to a child with empty parent
		0: {
			pre:   nil,
			child: ops,
			queries: []rangeQuery{
				// forward: no, start, finish, both limits
				{nil, nil, false, expect},
				{expect[10].Key, nil, false, expect[10:]},
				{nil, expect[Size-8].Key, false, expect[:Size-8]},
				{expect[17].Key, expect[28].Key, false, expect[17:28]},

				// reverse: no, start, finish, both limits
				{nil, nil, true, reverse(expect)},
				{expect[34].Key, nil, true, reverse(expect[34:])},
				{nil, expect[19].Key, true, reverse(expect[:19])},
				{expect[6].Key, expect[26].Key, true, reverse(expect[6:26])},
			},
		},
		// iterator combines child and parent
		1: {
			pre:   parentOps,
			child: ops,
			queries: []rangeQuery{
				// forward: no, start, finish, both limits
				{nil, nil, false, both},
				{both[10].Key, nil, false, both[10:]},
				{nil, both[Size-8].Key, false, both[:Size-8]},
				{both[17].Key, both[28].Key, false, both[17:28]},

				// reverse: no, start, finish, both limits
				{nil, nil, true, reverse(both)},
				{both[34].Key, nil, true, reverse(both[34:])},
				{nil, both[19].Key, true, reverse(both[:19])},
				{both[6].Key, both[26].Key, true, reverse(both[6:26])},
			},
		},
	}

	for i, tc := range cases {
		msg := fmt.Sprintf("FuzzBTreeCacheIterator: %d", i)
		base := makeBase()
		tc.verify(t, base, msg)
	}
}

// TestConflictBTreeCacheIterator makes sure the basic iterator
// works. Includes random deletes, but not nested iterators.
func TestConflictBTreeCacheIterator(t *testing.T) {
	const Size = 50
	const DeleteCount = 20

	ms := randModels(6, 20, 100)
	a, a2, b, b2, c, d := ms[0], ms[1], ms[2], ms[3], ms[4], ms[5]
	// a2, b2 have same keys, different values
	a2.Key = a.Key
	b2.Key = b.Key

	// toSet := randModels(Size, 8, 40)
	// toDel := randModels(DeleteCount, 8, 40)
	// expect := sortModels(toSet)
	// ops := append(
	// 	makeSetOps(toSet),
	// 	makeDelOps(toDel)...)

	// parentSet := randModels(Size, 8, 40)
	// parentDel := randModels(DeleteCount, 8, 40)
	// parentOps := append(
	// 	makeSetOps(parentSet),
	// 	makeDelOps(parentDel)...)

	// both := sortModels(append(toSet, parentSet...))

	expect0 := sortModels([]Model{a, b, c})
	expect1 := sortModels([]Model{a2, b2, c, d})
	expect2 := []Model{c}

	cases := [...]iterCase{
		// iterate in child only
		0: {
			child: makeSetOps(a, b, c),
			queries: []rangeQuery{
				// query for the values in child
				{nil, nil, false, expect0},
				{expect0[1].Key, expect0[2].Key, false, expect0[1:2]},

				{nil, nil, true, reverse(expect0)},
			},
		},
		// iterate over parent only
		1: {
			pre: makeSetOps(a, b, c),
			queries: []rangeQuery{
				// query for the values in child
				{nil, nil, false, expect0},
				{expect0[1].Key, expect0[2].Key, false, expect0[1:2]},

				{nil, nil, true, reverse(expect0)},
			},
		},
		// simple combination
		2: {
			pre:   makeSetOps(a, b),
			child: makeSetOps(c),
			queries: []rangeQuery{
				// query for the values in child
				{nil, nil, false, expect0},
				{expect0[1].Key, expect0[2].Key, false, expect0[1:2]},

				{nil, nil, true, reverse(expect0)},
			},
		},
		// overwrite data should show child data
		3: {
			pre:   makeSetOps(a, b, c),
			child: makeSetOps(a2, b2, d),
			queries: []rangeQuery{
				// query for the values in child
				{nil, nil, false, expect1},
				{expect1[1].Key, expect1[3].Key, false, expect1[1:3]},

				{nil, nil, true, reverse(expect1)},
			},
		},
		// overwrite data should show child data
		4: {
			pre:   makeSetOps(a, c, d),
			child: makeDelOps(a, b, d),
			queries: []rangeQuery{
				// query all should find just one, skip delete
				{nil, nil, false, expect2},
				// query cuts off at actual value, should be empty
				{nil, c.Key, false, nil},
			},
		},
	}

	for i, tc := range cases {
		msg := fmt.Sprintf("ConflictBTreeCacheIterator: %d", i)
		base := makeBase()
		tc.verify(t, base, msg)
	}
}

//--------------------- lots of helpers ------------------

func randBytes(length int) []byte {
	res := make([]byte, length)
	rand.Read(res)
	return res
}

// randKeys returns a slice of count keys, all of a given size
func randKeys(count, size int) [][]byte {
	res := make([][]byte, count)
	for i := 0; i < count; i++ {
		res[i] = randBytes(size)
	}
	return res
}

// randModels produces a random set of models
func randModels(count, keySize, valueSize int) []Model {
	models := make([]Model, count)
	for i := 0; i < count; i++ {
		models[i].Key = randBytes(keySize)
		models[i].Value = randBytes(valueSize)
	}
	return models
}

////////////////////////////////////////////////////
// helper methods to check queries

// iterCase is a test case for iteration
type iterCase struct {
	pre     []Op
	child   []Op
	queries []rangeQuery
}

func (i iterCase) verify(t *testing.T, base store.CacheableKVStore, msg string) {
	for _, op := range i.pre {
		op.Apply(base)
	}

	child := base.CacheWrap()
	for _, op := range i.child {
		op.Apply(base)
	}

	for j, q := range i.queries {
		jmsg := fmt.Sprintf("%s (%d)", msg, j)
		q.check(t, child, jmsg)
	}
}

// range query checks the results of iteration
type rangeQuery struct {
	start    []byte
	end      []byte
	reverse  bool
	expected []Model
}

func (q rangeQuery) check(t *testing.T, kv store.KVStore, msg string) {
	var iter store.Iterator
	if q.reverse {
		iter = kv.ReverseIterator(q.start, q.end)
	} else {
		iter = kv.Iterator(q.start, q.end)
	}
	verifyIterator(t, q.expected, iter, msg)
}

func verifyIterator(t *testing.T, models []Model, iter store.Iterator, msg string) {
	// make sure proper iteration works
	for i := 0; i < len(models); i++ {
		require.True(t, iter.Valid(), msg)
		assert.Equal(t, models[i].Key, iter.Key(), msg)
		assert.Equal(t, models[i].Value, iter.Value(), msg)
		iter.Next()
	}
	assert.False(t, iter.Valid())
	iter.Close()
}

// reverse returns a copy of the slice with elements in reverse order
func reverse(models []Model) []Model {
	max := len(models)
	res := make([]Model, max)
	for i := 0; i < max; i++ {
		res[i] = models[max-1-i]
	}
	return res
}

// sortModels returns a copy of the models sorted by key
func sortModels(models []Model) []Model {
	res := make([]Model, len(models))
	copy(res, models)
	// sort by key
	sort.Slice(res, func(i, j int) bool {
		return bytes.Compare(res[i].Key, res[j].Key) < 0
	})
	return res
}

func makeSetOps(ms ...Model) []Op {
	res := make([]Op, len(ms))
	for i, m := range ms {
		res[i] = store.SetOp(m.Key, m.Value)
	}
	return res
}

func makeDelOps(ms ...Model) []Op {
	res := make([]Op, len(ms))
	for i, m := range ms {
		res[i] = store.DelOp(m.Key)
	}
	return res
}
