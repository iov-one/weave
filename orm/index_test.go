package orm

import (
	"errors"
	"fmt"
	"testing"

	"github.com/confio/weave/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// simple indexer for Counter
func count(obj Object) ([]byte, error) {
	if obj == nil {
		return nil, errors.New("Cannot take index of nil")
	}
	cntr, ok := obj.Value().(*Counter)
	if !ok {
		return nil, errors.New("Can only take index of Counter")
	}
	// big-endian encoded int64
	return encodeSequence(cntr.Count), nil
}

func TestCounterIndex(t *testing.T) {
	multi := NewIndex("likes", count, false, nil)
	uniq := NewIndex("magic", count, true, nil)

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
		idx        Index
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
	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			idx := tc.idx
			err := idx.Update(db, tc.prev, tc.next)
			if tc.isError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tc.getLike != nil {
				res, err := idx.GetLike(db, tc.getLike)
				require.NoError(t, err)
				assert.EqualValues(t, tc.likeRes, res)
			}
			if tc.getAt != nil {
				res, err := idx.GetAt(db, tc.getAt)
				require.NoError(t, err)
				assert.EqualValues(t, tc.atRes, res)
			}
		})
	}
}

// simple indexer for MultiRef
// return first value (if any), or nil
func first(obj Object) ([]byte, error) {
	if obj == nil {
		return nil, errors.New("Cannot take index of nil")
	}
	multi, ok := obj.Value().(*MultiRef)
	if !ok {
		return nil, errors.New("Can only take index of MultiRef")
	}
	if len(multi.Refs) == 0 {
		return nil, nil
	}
	return multi.Refs[0], nil
}

func makeRefObj(key []byte, values ...[]byte) Object {
	value := &MultiRef{
		Refs: values,
	}
	return NewSimpleObj(key, value)
}

func checkNil(t *testing.T, objs ...Object) {
	for _, obj := range objs {
		bz, err := first(obj)
		require.NoError(t, err)
		require.Equal(t, 0, len(bz))
	}
}

// TestNullableIndex ensures we don't write indexes for nil values
// is that all wanted??
func TestNullableIndex(t *testing.T) {
	uniq := NewIndex("no-null", first, true, nil)

	// some keys to use
	k1 := []byte("abc")
	k2 := []byte("def")
	k3 := []byte("xyz")
	v1 := []byte("foo")
	v2 := []byte("bar")

	// o1 and o3 conflict
	o1 := makeRefObj(k1, v1, v2)
	o1a := makeRefObj(k1, v1)
	o2 := makeRefObj(k2, v2, v1)
	o3 := makeRefObj(k3, v1)

	// no nils should conflict
	n1 := makeRefObj(k1)
	n2 := makeRefObj(k2, []byte{}, v1)
	n3 := makeRefObj(k3, nil, v1)
	checkNil(t, n1, n2, n3)

	cases := []struct {
		setup      []Object // insert these first before test
		prev, next Object   // check for error
		isError    bool     // check insert result
	}{
		// make sure test works with non-nil objects
		0: {[]Object{o1}, nil, o2, false},
		1: {[]Object{o1, o2}, nil, o3, true},
		2: {[]Object{o1, o2}, o1, o1a, false},
		// make sure nil doens't cause conflicts
		3: {[]Object{}, nil, n1, false},
		4: {[]Object{n1}, nil, n2, false},
		5: {[]Object{n1}, nil, n3, false},
		6: {[]Object{o1, n1, o2}, nil, n2, false},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			db := store.MemStore()
			for _, init := range tc.setup {
				err := uniq.Update(db, nil, init)
				require.NoError(t, err)
			}

			err := uniq.Update(db, tc.prev, tc.next)
			if tc.isError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}

}
