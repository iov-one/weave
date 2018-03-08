package orm

import (
	"fmt"
	"testing"

	"github.com/confio/weave/store"
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
			res := incrementN(s, db, tc.add)
			assert.Equal(t, tc.expect, res)
		})
	}

}
