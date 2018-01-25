package store

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeBase returns the base layer
//
// If you want to test a different kvstore implementation
// you can copy most of these tests and change makeBase.
// Once that passes, customize and extend as you wish
func makeBase() CacheableKVStore {
	// devnull is a black hole... just to keep our types proper
	devnull := BTreeCacheable{EmptyKVStore{}}

	// base is the root of our data, we can layer on top and
	// all queries should work
	base := devnull.CacheWrap()

	return base
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
		parentOps     []op
		childOps      []op
		parentQueries []Model // Key is what we query, Value is what we espect
		childQueries  []Model // Key is what we query, Value is what we espect
	}{
		// overwrite one, delete another, add a third
		0: {
			[]op{setOp(ks[1], vs[1]), setOp(ks[2], vs[2])},
			[]op{setOp(ks[1], vs[11]), setOp(ks[3], vs[7]), delOp(ks[2])},
			[]Model{pair(ks[1], vs[1]), pair(ks[2], vs[2]), pair(ks[3], nil)},
			[]Model{pair(ks[1], vs[11]), pair(ks[2], nil), pair(ks[3], vs[7])},
		},
	}

	for i, tc := range cases {
		parent := makeBase()
		for _, op := range tc.parentOps {
			op.apply(parent)
		}

		child := parent.CacheWrap()
		for _, op := range tc.childOps {
			op.apply(child)
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

// TestSliceIterator makes sure the basic slice iterator works
func TestSliceIterator(t *testing.T) {
	const Size = 10

	ks := randKeys(Size, 8)
	vs := randKeys(Size, 40)

	models := make([]Model, Size)
	for i := 0; i < Size; i++ {
		models[i].Key = ks[i]
		models[i].Value = vs[i]
	}

	// make sure proper iteration works
	for iter, i := NewSliceIterator(models), 0; iter.Valid(); iter.Next() {
		assert.True(t, i < Size)
		assert.Equal(t, ks[i], iter.Key())
		assert.Equal(t, vs[i], iter.Value())
		i++
	}

	// iterator is invalid after close
	trash := NewSliceIterator(models)
	assert.True(t, trash.Valid())
	trash.Close()
	assert.False(t, trash.Valid())
}

// TestBTreeCacheBasicIterator makes sure the basic iterator
// works. Includes random deletes, but not nested iterators.
func TestBTreeCacheBasicIterator(t *testing.T) {
	const Size = 50
	const DeleteCount = 20

	toSet := randModels(Size, 8, 40)
	toDel := randModels(DeleteCount, 8, 40)
	expect := sortModels(toSet)
	ops := append(
		makeSetOps(toSet),
		makeDelOps(toDel)...)

	parentSet := randModels(Size, 8, 40)
	parentDel := randModels(DeleteCount, 8, 40)
	parentOps := append(
		makeSetOps(parentSet),
		makeDelOps(parentDel)...)

	both := sortModels(append(toSet, parentSet...))

	cases := [...]iterCase{
		// just write to a child with empty parent
		0: {
			pre:   nil,
			child: ops,
			queries: []rangeQuery{
				{nil, nil, false, expect},
				{expect[10].Key, nil, false, expect[10:]},
				{nil, expect[Size-8].Key, false, expect[:Size-8]},
				{expect[17].Key, expect[28].Key, false, expect[17:28]},

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
				{nil, nil, false, both},
				{both[10].Key, nil, false, both[10:]},
				{nil, both[Size-8].Key, false, both[:Size-8]},
				{both[17].Key, both[28].Key, false, both[17:28]},

				{nil, nil, true, reverse(both)},
				{both[34].Key, nil, true, reverse(both[34:])},
				{nil, both[19].Key, true, reverse(both[:19])},
				{both[6].Key, both[26].Key, true, reverse(both[6:26])},
			},
		},
	}

	for i, tc := range cases {
		msg := fmt.Sprintf("BTreeCacheBasicIterator: %d", i)
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
	pre     []op
	child   []op
	queries []rangeQuery
}

func (i iterCase) verify(t *testing.T, base CacheableKVStore, msg string) {
	for _, op := range i.pre {
		op.apply(base)
	}

	child := base.CacheWrap()
	for _, op := range i.child {
		op.apply(base)
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

func (q rangeQuery) check(t *testing.T, store KVStore, msg string) {
	var iter Iterator
	if q.reverse {
		iter = store.ReverseIterator(q.start, q.end)
	} else {
		iter = store.Iterator(q.start, q.end)
	}
	verifyIterator(t, q.expected, iter, msg)
}

func verifyIterator(t *testing.T, models []Model, iter Iterator, msg string) {
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
