package orm

import (
	"bytes"
	"reflect"
	"strconv"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestModelBucket(t *testing.T) {
	db := store.MemStore()

	b := NewModelBucket("cnts", &Counter{})

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

	b := NewModelBucket("cnts", &Counter{})

	// Using a nil key should cause the sequence ID to be used.
	key, err := b.Put(db, nil, &Counter{Count: 111})
	if err != nil {
		t.Fatalf("cannot save counter instance: %s", err)
	}
	if !bytes.Equal(key, weavetest.SequenceID(1)) {
		t.Fatalf("first sequence key should be 1, instead got %d", key)
	}

	// Inserting an entity with a key provided must not modify the ID
	// generation counter.
	if _, err := b.Put(db, []byte("mycnt"), &Counter{Count: 12345}); err != nil {
		t.Fatalf("cannot save counter instance: %s", err)
	}

	key, err = b.Put(db, nil, &Counter{Count: 222})
	if err != nil {
		t.Fatalf("cannot save counter instance: %s", err)
	}
	if !bytes.Equal(key, weavetest.SequenceID(2)) {
		t.Fatalf("second sequence key should be 2, instead got %d", key)
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
		QueryKey   string
		DestFn     func() ModelSlicePtr
		WantResPtr []*Counter
		WantRes    []Counter
		WantKeys   [][]byte
	}{
		"find none": {
			QueryKey:   "124089710947120",
			WantResPtr: nil,
			WantRes:    nil,
			WantKeys:   nil,
		},
		"find one": {
			QueryKey: "1",
			WantResPtr: []*Counter{
				{Count: 1001},
			},
			WantRes: []Counter{
				{Count: 1001},
			},
			WantKeys: [][]byte{
				weavetest.SequenceID(1),
			},
		},
		"find two": {
			QueryKey: "4",
			WantResPtr: []*Counter{
				{Count: 4001},
				{Count: 4002},
			},
			WantRes: []Counter{
				{Count: 4001},
				{Count: 4002},
			},
			WantKeys: [][]byte{
				weavetest.SequenceID(3),
				weavetest.SequenceID(4),
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()

			indexByBigValue := func(obj Object) ([][]byte, error) {
				c, ok := obj.Value().(*Counter)
				if !ok {
					return nil, errors.Wrapf(errors.ErrType, "%T", obj.Value())
				}
				// Index by the value, ignoring anything below 1k.
				raw := strconv.FormatInt(c.Count/1000, 10)
				return [][]byte{[]byte(raw)}, nil
			}

			// Use both native and compact index to test both
			// implementation integrations.
			b := NewModelBucket("cnts", &Counter{},
				WithNativeIndex("native", indexByBigValue),
				WithIndex("compact", indexByBigValue, false),
			)

			if _, err := b.Put(db, nil, &Counter{Count: 1001}); err != nil {
				t.Fatalf("cannot save counter instance: %s", err)
			}
			if _, err := b.Put(db, nil, &Counter{Count: 2001}); err != nil {
				t.Fatalf("cannot save counter instance: %s", err)
			}
			if _, err := b.Put(db, nil, &Counter{Count: 4001}); err != nil {
				t.Fatalf("cannot save counter instance: %s", err)
			}
			if _, err := b.Put(db, nil, &Counter{Count: 4002}); err != nil {
				t.Fatalf("cannot save counter instance: %s", err)
			}

			indexes := []string{"native", "compact"}
			for _, indexName := range indexes {
				t.Run(indexName, func(t *testing.T) {
					var dest []Counter
					keys, err := b.ByIndex(db, indexName, []byte(tc.QueryKey), &dest)
					if err != nil {
						t.Fatalf("unexpected error: %s", err)
					}
					assert.Equal(t, tc.WantKeys, keys)
					assert.Equal(t, tc.WantRes, dest)

					var destPtr []*Counter
					keys, err = b.ByIndex(db, indexName, []byte(tc.QueryKey), &destPtr)
					if err != nil {
						t.Fatalf("unexpected error: %s", err)
					}
					assert.Equal(t, tc.WantKeys, keys)
					assert.Equal(t, tc.WantResPtr, destPtr)
				})
			}
		})
	}
}

func TestModelBucketPutWrongModelType(t *testing.T) {
	db := store.MemStore()
	b := NewModelBucket("cnts", &Counter{})

	if _, err := b.Put(db, nil, &MultiRef{Refs: [][]byte{[]byte("foo")}}); !errors.ErrType.Is(err) {
		t.Fatalf("unexpected error when trying to store wrong model type value: %s", err)
	}
}

func TestModelBucketOneWrongModelType(t *testing.T) {
	db := store.MemStore()
	b := NewModelBucket("cnts", &Counter{})

	if _, err := b.Put(db, []byte("counter"), &Counter{Count: 1}); err != nil {
		t.Fatalf("cannot save counter instance: %s", err)
	}

	var ref MultiRef
	if err := b.One(db, []byte("counter"), &ref); !errors.ErrType.Is(err) {
		t.Fatalf("unexpected error when trying to get wrong model type value: %s", err)
	}
}

func TestModelBucketByIndexWrongModelType(t *testing.T) {
	db := store.MemStore()
	b := NewModelBucket("cnts", &Counter{},
		WithIndex("x", func(o Object) ([]byte, error) { return []byte("x"), nil }, false))

	if _, err := b.Put(db, []byte("counter"), &Counter{Count: 1}); err != nil {
		t.Fatalf("cannot save counter instance: %s", err)
	}

	var refs []MultiRef
	if _, err := b.ByIndex(db, "x", []byte("x"), &refs); !errors.ErrType.Is(err) {
		t.Fatalf("unexpected error when trying to find wrong model type value: %s: %v", err, refs)
	}

	var refsPtr []*MultiRef
	if _, err := b.ByIndex(db, "x", []byte("x"), &refsPtr); !errors.ErrType.Is(err) {
		t.Fatalf("unexpected error when trying to find wrong model type value: %s: %v", err, refs)
	}

	var refsPtrPtr []**MultiRef
	if _, err := b.ByIndex(db, "x", []byte("x"), &refsPtrPtr); !errors.ErrType.Is(err) {
		t.Fatalf("unexpected error when trying to find wrong model type value: %s: %v", err, refs)
	}
}

func TestModelBucketHas(t *testing.T) {
	db := store.MemStore()
	b := NewModelBucket("cnts", &Counter{})

	if _, err := b.Put(db, []byte("counter"), &Counter{Count: 1}); err != nil {
		t.Fatalf("cannot save counter instance: %s", err)
	}

	if err := b.Has(db, []byte("counter")); err != nil {
		t.Fatalf("an existing entity must return no error: %s", err)
	}

	if err := b.Has(db, nil); !errors.ErrNotFound.Is(err) {
		t.Fatalf("a nil key must return ErrNotFound: %s", err)
	}

	if err := b.Has(db, []byte("does-not-exist")); !errors.ErrNotFound.Is(err) {
		t.Fatalf("a non exists entity must return ErrNotFound: %s", err)
	}
}

func TestIterAll(t *testing.T) {
	type obj struct {
		Key   string
		Model Counter
	}

	cases := map[string]struct {
		Objs         []obj
		WantKeys     []string
		WantCounters []Counter
	}{
		"empty": {
			Objs:         nil,
			WantKeys:     nil,
			WantCounters: nil,
		},
		"single element": {
			Objs: []obj{
				{Key: "a", Model: Counter{Count: 1}},
			},
			WantKeys:     []string{"a"},
			WantCounters: []Counter{{Count: 1}},
		},
		"multiple elements": {
			Objs: []obj{
				{Key: "a", Model: Counter{Count: 1}},
				{Key: "c", Model: Counter{Count: 3}},
				{Key: "b", Model: Counter{Count: 2}},
			},
			WantKeys:     []string{"a", "b", "c"},
			WantCounters: []Counter{{Count: 1}, {Count: 2}, {Count: 3}},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()

			b := NewModelBucket("cnts", &Counter{})
			for i, o := range tc.Objs {
				if _, err := b.Put(db, []byte(o.Key), &o.Model); err != nil {
					t.Fatalf("%d: cannot put %q token: %s", i, o.Key, err)
				}
			}

			// Add some rubbish to the database, so that any
			// unexected result can be detected.
			db.Set([]byte{0}, []byte("xyz"))
			db.Set([]byte{255}, []byte("z"))
			db.Set([]byte("mystuff:abc"), []byte("mystuff"))

			keys, counters := consumeIterAll(t, db, IterAll("cnts"))
			if !reflect.DeepEqual(keys, tc.WantKeys) {
				t.Logf("want: %q", tc.WantKeys)
				t.Logf(" got: %q", keys)
				t.Error("unexpected iterator keys")
			}
			if !reflect.DeepEqual(counters, tc.WantCounters) {
				t.Logf("want: %+v", tc.WantCounters)
				t.Logf(" got: %+v", counters)
				t.Error("unexpected iterator values")
			}
		})
	}
}

func consumeIterAll(t testing.TB, db weave.ReadOnlyKVStore, it *ModelBucketIterator) ([]string, []Counter) {
	t.Helper()

	var (
		counters []Counter
		keys     []string
	)
	for {
		var c Counter
		switch key, err := it.Next(db, &c); {
		case err == nil:
			keys = append(keys, string(key))
			counters = append(counters, c)
		case errors.ErrIteratorDone.Is(err):
			return keys, counters
		default:
			t.Fatalf("next: %s", err)
		}
	}
}
