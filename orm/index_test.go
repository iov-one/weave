package orm

import (
	"bytes"
	stderrors "errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest/assert"
)

// newIndex constructs an index with single key Indexer.
// Indexer calculates the index for an object
// unique enforces a unique constraint on the index
// refKey calculates the absolute dbkey for a ref
func newIndex(name string, indexer Indexer, unique bool, refKey func([]byte) []byte) Index {
	return NewMultiKeyIndex(name, asMultiKeyIndexer(indexer), unique, refKey)
}

// simple indexer for Counter
func count(obj Object) ([]byte, error) {
	if obj == nil {
		return nil, stderrors.New("cannot take index of nil")
	}
	cntr, ok := obj.Value().(*Counter)
	if !ok {
		return nil, stderrors.New("can only take index of Counter")
	}
	// big-endian encoded int64
	return encodeSequence(cntr.Count), nil
}

func TestCounterSingleKeyIndex(t *testing.T) {
	multi := newIndex("likes", count, false, nil).(compactIndex)
	uniq := newIndex("magic", count, true, nil).(compactIndex)

	// some keys to use
	k1 := []byte("abc")
	k2 := []byte("def")
	k3 := []byte("xyz")

	o1 := NewSimpleObj(k1, NewCounter(5))
	o1a := NewSimpleObj(k1, NewCounter(7))
	o2 := NewSimpleObj(k2, NewCounter(7))
	o2a := NewSimpleObj(k2, NewCounter(9))
	o3 := NewSimpleObj(k3, NewCounter(9))
	o3a := NewSimpleObj(k3, NewCounter(5))

	e5 := encodeSequence(5)
	e7 := encodeSequence(7)
	e9 := encodeSequence(9)

	cases := []struct {
		idx        compactIndex
		prev, next Object // for Update
		isError    bool   // check Update result
		// if there was no error, and these are non-nil, try
		getLike Object
		likeRes [][]byte
		getAt   []byte
		atRes   [][]byte
	}{
		// we can only add things that make sense
		0: {multi, nil, nil, true, nil, nil, nil, nil},
		1: {multi, o1, nil, true, nil, nil, nil, nil},
		// insert works
		2: {multi, nil, o1, false, o1, [][]byte{k1}, e5, [][]byte{k1}},
		3: {multi, nil, o2, false, o2, [][]byte{k2}, e7, [][]byte{k2}},
		// insert same second time fails
		4: {multi, nil, o1, true, nil, nil, nil, nil},
		// remove not inserted fails
		5: {multi, o3, nil, true, nil, nil, nil, nil},
		// we can combine them (note keys sorted, not by insert time)
		6: {multi, o1, o1a, false, o1, nil, e7, [][]byte{k1, k2}},
		// add another one (note that primary key is not to search)
		7: {multi, nil, o3, false, o3, [][]byte{k3}, k3, nil},
		// move from one list to another
		8: {multi, o2, o2a, false, o2a, [][]byte{k2, k3}, e7, [][]byte{k1}},
		// remove works
		9:  {multi, o2a, nil, false, nil, nil, e9, [][]byte{k3}},
		10: {multi, o1a, nil, false, nil, nil, e7, nil},
		// leave with one object at key 5
		11: {multi, o3, o3a, false, o3, nil, e5, [][]byte{k3}},
		// uniq has no conflict with other bucket
		12: {uniq, nil, o1, false, nil, nil, e5, [][]byte{k1}},
		// but cannot add two at one location
		13: {uniq, nil, o3a, true, nil, nil, nil, nil},
		// add a second one
		14: {uniq, nil, o2, false, nil, nil, e7, [][]byte{k2}},
		// move that causes conflict fails
		15: {uniq, o1, o1a, true, nil, nil, nil, nil},
		// remove works
		16: {uniq, o2, nil, false, o2, nil, e5, [][]byte{k1}},
		// second remove fails
		17: {uniq, o2, nil, true, nil, nil, nil, nil},
		// now we can move it
		18: {uniq, o1, o1a, false, o1, nil, e7, [][]byte{k1}},
	}

	db := store.MemStore()
	for i, tc := range cases { // can not be converted into table tests easily as there is a dependency between cases
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			idx := tc.idx
			err := idx.Update(db, tc.prev, tc.next)
			if tc.isError {
				assert.Equal(t, true, err != nil)
				return
			}

			assert.Nil(t, err)
			if tc.getLike != nil {
				res, err := idx.Like(db, tc.getLike)
				assert.Nil(t, err)
				assert.Equal(t, tc.likeRes, res)
			}
			if tc.getAt != nil {
				res, err := consumeIteratorKeys(idx.Keys(db, tc.getAt))
				assert.Nil(t, err)
				assert.Equal(t, tc.atRes, res)
			}
		})
	}
}

func TestCounterMultiKeyIndex(t *testing.T) {
	uniq := NewMultiKeyIndex("unique", evenOddIndexer, true, nil).(compactIndex)

	specs := map[string]struct {
		index               compactIndex
		store               Object
		prev, next          Object
		expError            bool
		expKeys, expNotKeys [][]byte
	}{
		"update with all keys replaced": {
			index:      uniq,
			prev:       NewSimpleObj([]byte("my"), NewCounter(5)),
			next:       NewSimpleObj([]byte("my"), NewCounter(6)),
			expKeys:    [][]byte{encodeSequence(6), []byte("even")},
			expNotKeys: [][]byte{encodeSequence(5), []byte("odd")},
		},
		"update with 1 key updated only": {
			index:      uniq,
			prev:       NewSimpleObj([]byte("my"), NewCounter(6)),
			next:       NewSimpleObj([]byte("my"), NewCounter(8)),
			expKeys:    [][]byte{encodeSequence(8), []byte("even")},
			expNotKeys: [][]byte{encodeSequence(6)},
		},
		"insert": {
			index:   uniq,
			next:    NewSimpleObj([]byte("my"), NewCounter(6)),
			expKeys: [][]byte{encodeSequence(6), []byte("even")},
		},
		"delete": {
			index:      uniq,
			prev:       NewSimpleObj([]byte("my"), NewCounter(5)),
			expNotKeys: [][]byte{encodeSequence(5), []byte("odd")},
		},
		"update with unique constraint fail": {
			index:    uniq,
			store:    NewSimpleObj([]byte("even"), NewCounter(8)),
			prev:     NewSimpleObj([]byte("my"), NewCounter(5)),
			next:     NewSimpleObj([]byte("my"), NewCounter(6)),
			expError: true,
		},
		"update without unique constraint": {
			index:    NewMultiKeyIndex("multi", evenOddIndexer, false, nil).(compactIndex),
			store:    NewSimpleObj([]byte("even"), NewCounter(8)),
			prev:     NewSimpleObj([]byte("my"), NewCounter(5)),
			next:     NewSimpleObj([]byte("my"), NewCounter(6)),
			expKeys:  [][]byte{encodeSequence(6), []byte("even")},
			expError: false,
		},
		"id mismatch": {
			index:    uniq,
			prev:     NewSimpleObj([]byte("my"), NewCounter(5)),
			next:     NewSimpleObj([]byte("bar"), NewCounter(7)),
			expError: true,
		},
		"both nil": {
			index:    uniq,
			expError: true,
		},
	}

	for testName, spec := range specs {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()

			// given
			idx := spec.index
			for _, o := range []Object{spec.store, spec.prev} {
				if o == nil {
					continue
				}
				keys, _ := idx.index(o)
				for _, key := range keys {
					assert.Nil(t, idx.insert(db, key, o.Key()))
				}
			}
			// when
			err := idx.Update(db, spec.prev, spec.next)

			// then
			if spec.expError {
				assert.Equal(t, true, err != nil)
			} else {
				assert.Nil(t, err)
			}
			for _, k := range spec.expKeys {
				// and index keys exists
				pks, err := consumeIteratorKeys(idx.Keys(db, k))
				assert.Nil(t, err)
				// with proper pk
				if idx.unique {
					assert.Equal(t, [][]byte{[]byte("my")}, pks)
				} else {
					var found bool
					for i := range pks {
						if exp, got := []byte("my"), pks[i]; bytes.Equal(exp, got) {
							found = true
							break
						}
					}
					assert.Equal(t, true, found)
				}
			}
			// and previous index keys don't exist anymore
			for _, k := range spec.expNotKeys {
				pks, err := consumeIteratorKeys(idx.Keys(db, k))
				assert.Nil(t, err)
				assert.Nil(t, pks)
			}
		})
	}
}

func TestGetLikeWithMultiKeyIndex(t *testing.T) {
	db := store.MemStore()
	idx := NewMultiKeyIndex("multi", evenOddIndexer, false, nil).(compactIndex)

	persistentObjects := []Object{
		NewSimpleObj([]byte("firstOdd"), NewCounter(5)),
		NewSimpleObj([]byte("secondOdd"), NewCounter(7)),
		NewSimpleObj([]byte("even"), NewCounter(8)),
	}
	for _, o := range persistentObjects {
		keys, _ := idx.index(o)
		for _, key := range keys {
			assert.Nil(t, idx.insert(db, key, o.Key()))
		}
	}

	specs := map[string]struct {
		source Object
		expPKs [][]byte
	}{
		"any odd counter value matches all other odd entries": {
			source: NewSimpleObj([]byte("anyOdd"), NewCounter(9)),
			expPKs: [][]byte{[]byte("firstOdd"), []byte("secondOdd")},
		},
		"obj key does not matter with this indexer": {
			source: NewSimpleObj([]byte("firstOdd"), NewCounter(5)),
			expPKs: [][]byte{[]byte("firstOdd"), []byte("secondOdd")},
		},
		"even counter value matches other even objects": {
			source: NewSimpleObj([]byte("even"), NewCounter(8)),
			expPKs: [][]byte{[]byte("even")},
		},
		"obj key does not matter here, too": {
			source: NewSimpleObj([]byte("anotherEven"), NewCounter(10)),
			expPKs: [][]byte{[]byte("even")},
		},
	}
	for testName, spec := range specs {
		t.Run(testName, func(t *testing.T) {
			pks, err := idx.Like(db, spec.source)
			// then
			assert.Nil(t, err)
			assert.Equal(t, spec.expPKs, pks)
		})
	}
}

func evenOddIndexer(obj Object) ([][]byte, error) {
	cntr, _ := obj.Value().(*Counter)
	result := [][]byte{encodeSequence(cntr.Count)}
	switch {
	case cntr.Count == 0:
	case cntr.Count%2 == 0:
		result = append(result, []byte("even"))
	default:
		result = append(result, []byte("odd"))
	}
	return result, nil
}

// simple indexer for MultiRef
// return first value (if any), or nil
func first(obj Object) ([]byte, error) {
	if obj == nil {
		return nil, stderrors.New("Cannot take index of nil")
	}
	multi, ok := obj.Value().(*MultiRef)
	if !ok {
		return nil, stderrors.New("Can only take index of MultiRef")
	}
	if len(multi.Refs) == 0 {
		return nil, nil
	}
	return multi.Refs[0], nil
}

func checkNil(t *testing.T, objs ...Object) {
	for _, obj := range objs {
		bz, err := first(obj)
		assert.Nil(t, err)
		assert.Equal(t, 0, len(bz))
	}
}

// TestNullableIndex ensures we don't write indexes for nil values
// is that all wanted??
func TestNullableIndex(t *testing.T) {
	// some keys to use
	k1 := []byte("abc")
	k2 := []byte("def")
	k3 := []byte("xyz")
	v1 := []byte("foo")
	v2 := []byte("bar")

	makeRefObj := func(key []byte, values ...[]byte) Object {
		return NewSimpleObj(key, &MultiRef{
			Refs: values,
		})
	}

	// o1 and o3 conflict (different key but v1 at pos 1)
	o1 := makeRefObj(k1, v1, v2)
	o1a := makeRefObj(k1, v1)
	o2 := makeRefObj(k2, v2, v1)
	o3 := makeRefObj(k3, v1)

	// no nils should conflict
	n1 := makeRefObj(k1)
	n1a := makeRefObj(k1, []byte{}, v2)
	n2 := makeRefObj(k2, []byte{}, v1)
	n3 := makeRefObj(k3, nil, v1)
	checkNil(t, n1, n2, n3)

	cases := map[string]struct {
		setup      []Object // insert these first before test
		prev, next Object   // check for error
		isError    bool     // check insert result
	}{
		"add non existing": {
			[]Object{o1}, nil, o2, false},
		"non unique values must be rejected": {
			[]Object{o1, o2}, nil, o3, true},
		"update value for existing key": {
			[]Object{o1, o2}, o1, o1a, false},
		"nil doesn't cause conflicts: allow index nil value": {
			[]Object{}, nil, n1, false},
		"nil doesn't cause conflicts: allow index empty bytes value": {
			[]Object{n1}, nil, n2, false},
		"nil doesn't cause conflicts: constraint": {
			[]Object{n1}, nil, n3, false},
		"nil doesn't cause conflicts: can add empty bytes value": {
			[]Object{o1, n1, o2}, nil, n2, false},
		"can update nil value": {
			[]Object{n1, n2}, n1, n1a, false},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			uniq := newIndex("no-null", first, true, nil)
			db := store.MemStore()
			for _, init := range tc.setup {
				err := uniq.Update(db, nil, init)
				assert.Nil(t, err)
			}
			// when
			err := uniq.Update(db, tc.prev, tc.next)
			if tc.isError {
				assert.Equal(t, true, err != nil)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestDeduplicatePKList(t *testing.T) {
	specs := map[string]struct {
		src, exp []string
	}{
		"empty":                             {src: []string{}, exp: []string{}},
		"duplicate dropped":                 {src: []string{"a", "a"}, exp: []string{"a"}},
		"duplicate at the start":            {src: []string{"a", "a", "b"}, exp: []string{"a", "b"}},
		"duplicate at the end":              {src: []string{"a", "b", "a"}, exp: []string{"a", "b"}},
		"two duplicates":                    {src: []string{"a", "b", "a", "b"}, exp: []string{"a", "b"}},
		"order preserved without duplicate": {src: []string{"a", "b", "c", "b", "d"}, exp: []string{"a", "b", "c", "d"}},
		"works with nil":                    {src: nil, exp: nil},
	}
	for testName, spec := range specs {
		t.Run(testName, func(t *testing.T) {
			assert.Equal(t, toBytes(spec.exp), deduplicate(toBytes(spec.src)))
		})
	}
}

func TestSubtract(t *testing.T) {
	specs := map[string]struct {
		src, sub, exp []string
	}{
		"all empty":            {src: []string{}, sub: []string{}, exp: []string{}},
		"single existing":      {src: []string{"a", "b", "c"}, sub: []string{"a"}, exp: []string{"b", "c"}},
		"multiple existing":    {src: []string{"a", "b", "c"}, sub: []string{"b", "c"}, exp: []string{"a"}},
		"non existing ignored": {src: []string{"a", "b", "c"}, sub: []string{"b", "d"}, exp: []string{"a", "c"}},
		"nil as sub":           {src: []string{"a"}, sub: nil, exp: []string{"a"}},
		"sub from nil":         {src: nil, sub: []string{"a"}, exp: nil},
		"all nil":              {src: nil, sub: nil, exp: nil},
	}
	for testName, spec := range specs {
		t.Run(testName, func(t *testing.T) {
			assert.Equal(t, toBytes(spec.exp), subtract(toBytes(spec.src), toBytes(spec.sub)))
		})
	}
}

func toBytes(s []string) [][]byte {
	if s == nil {
		return nil
	}
	source := make([][]byte, len(s))
	for i, v := range s {
		source[i] = []byte(v)
	}
	return source
}

func TestNativeIndexPacking(t *testing.T) {
	cases := map[string][][]byte{
		"empty":                [][]byte{},
		"one empty element":    [][]byte{[]byte{}},
		"three empty elements": [][]byte{[]byte{}, []byte{}, []byte{}},
		"one non empty element": [][]byte{
			[]byte("foo"),
		},
		"two non empty elements": [][]byte{
			[]byte("a"),
			[]byte("a very long value that is below 255 characters"),
		},
		"three non empty elements": [][]byte{
			[]byte("foo"),
			[]byte("bar"),
			[]byte("baz"),
		},
		"mixture of empty and non empty": [][]byte{
			[]byte("non empty value"),
			[]byte{},
			[]byte("another non empty value"),
			[]byte{},
			[]byte{},
			[]byte{},
			[]byte("not empty"),
			[]byte{},
		},
	}

	for testName, chunks := range cases {
		t.Run(testName, func(t *testing.T) {
			packed, err := packNativeIdxKey(chunks)
			if err != nil {
				t.Fatalf("cannot pack: %s", err)
			}
			unpacked, err := unpackNativeIdxKey(packed)
			if err != nil {
				t.Fatalf("cannot unpack: %s", err)
			}
			if !reflect.DeepEqual(unpacked, chunks) {
				t.Logf("packed: %x %q", packed, packed)
				t.Fatalf("data malformed during serialization: %q", unpacked)
			}
		})
	}
}

func TestCompactIndexImplementation(t *testing.T) {
	testIndexImplementation(t, func(fn MultiKeyIndexer) Index {
		return NewMultiKeyIndex("myindex", fn, false, func(b []byte) []byte { return b })
	})
}

func TestNativeIndexImplementation(t *testing.T) {
	testIndexImplementation(t, func(fn MultiKeyIndexer) Index {
		return NewNativeIndex("myindex", fn)
	})
}

func testIndexImplementation(t *testing.T, newIdx func(MultiKeyIndexer) Index) {
	valueIdx := func(o Object) ([][]byte, error) {
		c := o.Value().(*Counter).Count
		return [][]byte{[]byte(fmt.Sprint(c))}, nil
	}
	idx := newIdx(valueIdx)

	// Definition of a single Update method call and expected result.
	type updateCall struct {
		prev    Object
		next    Object
		wantErr *errors.Error
	}

	// Definition of a single Keys method call and expected results.
	type keysCall struct {
		value    []byte
		wantKeys []string // []string and not [][]byte for nicer UI
	}

	cases := map[string]struct {
		idx     Index
		updates []updateCall
		keys    []keysCall
	}{
		"no results found": {
			idx:     idx,
			updates: []updateCall{},
			keys: []keysCall{
				{value: []byte("random-value"), wantKeys: nil},
			},
		},
		"a single item": {
			idx: idx,
			updates: []updateCall{
				{prev: nil, next: NewSimpleObj([]byte("one"), &Counter{Count: 1})},
			},
			keys: []keysCall{
				{value: []byte("1"), wantKeys: []string{"one"}},
				{value: []byte("unindexed-value"), wantKeys: nil},
			},
		},
		"two items, both with the same index value": {
			idx: idx,
			updates: []updateCall{
				{prev: nil, next: NewSimpleObj([]byte("first"), &Counter{Count: 1})},
				{prev: nil, next: NewSimpleObj([]byte("second"), &Counter{Count: 1})},
			},
			keys: []keysCall{
				{value: []byte("1"), wantKeys: []string{"first", "second"}},
				{value: []byte("unindexed-value"), wantKeys: nil},
			},
		},
		"two items, each with a different index value": {
			idx: idx,
			updates: []updateCall{
				{prev: nil, next: NewSimpleObj([]byte("one"), &Counter{Count: 1})},
				{prev: nil, next: NewSimpleObj([]byte("two"), &Counter{Count: 2})},
			},
			keys: []keysCall{
				{value: []byte("1"), wantKeys: []string{"one"}},
				{value: []byte("2"), wantKeys: []string{"two"}},
				{value: []byte("unindexed-value"), wantKeys: nil},
			},
		},
		"many items, some with similar index value": {
			idx: idx,
			updates: []updateCall{
				{prev: nil, next: NewSimpleObj([]byte("a"), &Counter{Count: 1})},
				{prev: nil, next: NewSimpleObj([]byte("b"), &Counter{Count: 2})},
				{prev: nil, next: NewSimpleObj([]byte("c"), &Counter{Count: 2})},
				{prev: nil, next: NewSimpleObj([]byte("d"), &Counter{Count: 2})},
				{prev: nil, next: NewSimpleObj([]byte("e"), &Counter{Count: 3})},
			},
			keys: []keysCall{
				{value: []byte("1"), wantKeys: []string{"a"}},
				{value: []byte("2"), wantKeys: []string{"b", "c", "d"}},
				{value: []byte("3"), wantKeys: []string{"e"}},
				{value: []byte("unindexed-value"), wantKeys: nil},
			},
		},
		"deleting an item from an index": {
			idx: idx,
			updates: []updateCall{
				{prev: nil, next: NewSimpleObj([]byte("one"), &Counter{Count: 1})},
				{prev: nil, next: NewSimpleObj([]byte("two"), &Counter{Count: 2})},
				{prev: NewSimpleObj([]byte("two"), &Counter{Count: 2}), next: nil},
			},
			keys: []keysCall{
				{value: []byte("1"), wantKeys: []string{"one"}},
				{value: []byte("2"), wantKeys: nil},
			},
		},
		"reindexing an item with a different value": {
			idx: idx,
			updates: []updateCall{
				{prev: nil, next: NewSimpleObj([]byte("two"), &Counter{Count: 1})},
				{prev: nil, next: NewSimpleObj([]byte("one"), &Counter{Count: 1})},
				{prev: NewSimpleObj([]byte("two"), &Counter{Count: 1}), next: NewSimpleObj([]byte("two"), &Counter{Count: 2})},
			},
			keys: []keysCall{
				{value: []byte("1"), wantKeys: []string{"one"}},
				{value: []byte("2"), wantKeys: []string{"two"}},
				{value: []byte("unindexed-value"), wantKeys: nil},
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()

			for i, u := range tc.updates {
				if err := tc.idx.Update(db, u.prev, u.next); !u.wantErr.Is(err) {
					t.Fatalf("%d update: want %q error, got %q", i, u.wantErr, err)
				}
			}

			for i, k := range tc.keys {
				keys, err := iteratorKeys(tc.idx.Keys(db, k.value))
				if err != nil {
					t.Fatalf("%d iterator keys failed: %s", i, err)
				}

				var want [][]byte
				for _, k := range k.wantKeys {
					want = append(want, []byte(k))
				}
				if !reflect.DeepEqual(keys, want) {
					t.Logf("want keys: %q", want)
					t.Logf(" got keys: %q", keys)
					t.Errorf("%d keys call returned unexpected keys for value %q", i, k.value)
				}
			}
		})
	}
}

func iteratorKeys(it weave.Iterator) ([][]byte, error) {
	defer it.Release()

	var res [][]byte
	for {
		switch key, _, err := it.Next(); {
		case err == nil:
			res = append(res, key)
		case errors.ErrIteratorDone.Is(err):
			return res, nil
		default:
			return res, err
		}
	}
}
