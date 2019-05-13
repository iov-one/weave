package orm

import (
	"strconv"
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestModelBucket(t *testing.T) {
	db := store.MemStore()

	obj := NewSimpleObj(nil, &Counter{})
	objBucket := NewBucket("cnts", obj)

	b := NewModelBucket(objBucket)

	if _, err := b.Put(db, []byte("c1"), &Counter{Count: 1}); err != nil {
		t.Fatalf("cannot save counter instance: %s", err)
	}

	var c1 Counter
	if err := b.One(db, []byte("c1"), &c1); err != nil {
		t.Fatalf("cannot get c1 counter: %s", err)
	}
	if c1.Count != 1 {
		t.Fatalf("unexpected counter state: %d", c1)
	}

	if err := b.Delete(db, []byte("c1")); err != nil {
		t.Fatalf("cannot delete c1 counter: %s", err)
	}
	if err := b.Delete(db, []byte("unknown")); !errors.ErrNotFound.Is(err) {
		t.Fatalf("unexpected error when deleting unexisting instance: %s", err)
	}
	if err := b.One(db, []byte("c1"), &c1); !errors.ErrNotFound.Is(err) {
		t.Fatalf("unexpected error for an unknown model get: %s", err)
	}
}

func TestModelBucketPutSequence(t *testing.T) {
	db := store.MemStore()

	obj := NewSimpleObj(nil, &Counter{})
	objBucket := NewBucket("cnts", obj)

	b := NewModelBucket(objBucket)

	// Using a nil key should cause the sequence ID to be used.
	if _, err := b.Put(db, nil, &Counter{Count: 111}); err != nil {
		t.Fatalf("cannot save counter instance: %s", err)
	}

	// Inserting an entity with a key provided must not modify the ID
	// generation counter.
	if _, err := b.Put(db, []byte("mycnt"), &Counter{Count: 12345}); err != nil {
		t.Fatalf("cannot save counter instance: %s", err)
	}

	if _, err := b.Put(db, nil, &Counter{Count: 222}); err != nil {
		t.Fatalf("cannot save counter instance: %s", err)
	}

	var c1 Counter
	if err := b.One(db, weavetest.SequenceID(1), &c1); err != nil {
		t.Fatalf("cannot get first counter: %s", err)
	}
	if c1.Count != 111 {
		t.Fatalf("unexpected counter state: %d", c1)
	}

	var c2 Counter
	if err := b.One(db, weavetest.SequenceID(2), &c2); err != nil {
		t.Fatalf("cannot get first counter: %s", err)
	}
	if c2.Count != 222 {
		t.Fatalf("unexpected counter state: %d", c2)
	}
}

func TestModelBucketByIndex(t *testing.T) {
	cases := map[string]struct {
		IndexName  string
		QueryKey   string
		DestFn     func() ModelSlicePtr
		WantErr    *errors.Error
		WantResPtr []*Counter
		WantRes    []Counter
	}{
		"find none": {
			IndexName:  "value",
			QueryKey:   "124089710947120",
			WantErr:    nil,
			WantResPtr: nil,
			WantRes:    nil,
		},
		"find one": {
			IndexName: "value",
			QueryKey:  "1111",
			WantErr:   nil,
			WantResPtr: []*Counter{
				&Counter{Count: 1111},
			},
			WantRes: []Counter{
				Counter{Count: 1111},
			},
		},
		"find two": {
			IndexName: "value",
			QueryKey:  "4444",
			WantErr:   nil,
			WantResPtr: []*Counter{
				&Counter{Count: 4444},
				&Counter{Count: 4444},
			},
			WantRes: []Counter{
				Counter{Count: 4444},
				Counter{Count: 4444},
			},
		},
		"non existing index name": {
			IndexName: "xyz",
			WantErr:   ErrInvalidIndex,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()

			indexByValue := func(obj Object) ([]byte, error) {
				c, ok := obj.Value().(*Counter)
				if !ok {
					return nil, errors.Wrapf(errors.ErrType, "%T", obj.Value())
				}
				raw := strconv.FormatInt(c.Count, 10)
				return []byte(raw), nil
			}
			objBucket := NewBucket("cnts", NewSimpleObj(nil, &Counter{})).
				WithIndex("value", indexByValue, false)
			b := NewModelBucket(objBucket)

			if _, err := b.Put(db, nil, &Counter{Count: 4444}); err != nil {
				t.Fatalf("cannot save counter instance: %s", err)
			}
			if _, err := b.Put(db, nil, &Counter{Count: 4444}); err != nil {
				t.Fatalf("cannot save counter instance: %s", err)
			}
			if _, err := b.Put(db, nil, &Counter{Count: 1111}); err != nil {
				t.Fatalf("cannot save counter instance: %s", err)
			}
			if _, err := b.Put(db, nil, &Counter{Count: 99999}); err != nil {
				t.Fatalf("cannot save counter instance: %s", err)
			}

			var dest []Counter
			err := b.ByIndex(db, tc.IndexName, []byte(tc.QueryKey), &dest)
			if !tc.WantErr.Is(err) {
				t.Fatalf("unexpected error: %s", err)
			}
			assert.Equal(t, tc.WantRes, dest)

			var destPtr []*Counter
			err = b.ByIndex(db, tc.IndexName, []byte(tc.QueryKey), &destPtr)
			if !tc.WantErr.Is(err) {
				t.Fatalf("unexpected error: %s", err)
			}
			assert.Equal(t, tc.WantResPtr, destPtr)
		})
	}
}
