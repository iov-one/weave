package orm

import (
	"encoding/binary"
	"strconv"
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestSerialModelBucket(t *testing.T) {
	db := store.MemStore()

	b := NewSerialModelBucket("cnts", &CounterWithID{})

	key1 := []byte("c1")
	err := b.Put(db, &CounterWithID{ID: key1, Count: 1})
	assert.Nil(t, err)

	var c1 CounterWithID
	err = b.One(db, key1, &c1)
	assert.Nil(t, err)
	assert.Equal(t, int64(1), c1.Count)
	assert.Equal(t, key1, c1.GetID())

	err = b.Delete(db, key1)
	assert.Nil(t, err)
	if err := b.Delete(db, []byte("unknown")); !errors.ErrNotFound.Is(err) {
		t.Fatalf("unexpected error when deleting unexisting instance: %s", err)
	}
	if err := b.One(db, key1, &c1); !errors.ErrNotFound.Is(err) {
		t.Fatalf("unexpected error for an unknown model get: %s", err)
	}
}

func TestSerialModelBucketPutSequence(t *testing.T) {
	db := store.MemStore()

	b := NewSerialModelBucket("cnts", &CounterWithID{})

	// Using a nil key should cause the sequence ID to be used.
	cnt := CounterWithID{Count: 111}
	assert.Nil(t, cnt.GetID())
	err := b.Put(db, &cnt)
	assert.Nil(t, err)
	assert.Equal(t, cnt.GetID(), weavetest.SequenceID(1))

	// Inserting an entity with a key provided must not modify the ID
	// generation counter.
	err = b.Put(db, &CounterWithID{ID: []byte("mycnt"), Count: 12345})
	assert.Nil(t, err)

	cnt2 := CounterWithID{Count: 222}
	err = b.Put(db, &cnt2)
	assert.Nil(t, err)
	assert.Equal(t, cnt2.GetID(), weavetest.SequenceID(2))

	var c1 CounterWithID
	err = b.One(db, weavetest.SequenceID(1), &c1)
	assert.Nil(t, err)
	assert.Equal(t, weavetest.SequenceID(1), c1.GetID())
	assert.Equal(t, int64(111), c1.Count)

	var c2 CounterWithID
	err = b.One(db, weavetest.SequenceID(2), &c2)
	assert.Nil(t, err)
	assert.Equal(t, weavetest.SequenceID(2), c2.GetID())
	assert.Equal(t, int64(222), c2.Count)
}

func TestSerialModelBucketPrefixScan(t *testing.T) {
	db := store.MemStore()

	b := NewSerialModelBucket("cnts", &CounterWithID{})

	cnts := []CounterWithID{
		CounterWithID{Count: 1},
		CounterWithID{Count: 17},
		CounterWithID{Count: 11},
		CounterWithID{Count: 3},
	}

	for i := range cnts {
		// make sure we point to value in array, so this ID gets set
		err := b.Put(db, &cnts[i])
		assert.Nil(t, err)
	}

	var loaded CounterWithID
	iter, err := b.PrefixScan(db, nil, false)
	assert.Nil(t, err)

	err = iter.LoadNext(&loaded)
	assert.Nil(t, err)
	assert.Equal(t, cnts[0], loaded)

	err = iter.LoadNext(&loaded)
	assert.Nil(t, err)
	assert.Equal(t, cnts[1], loaded)

	iter.Release()

	// validate reverse also works
	iter, err = b.PrefixScan(db, nil, true)
	assert.Nil(t, err)
	err = iter.LoadNext(&loaded)
	assert.Nil(t, err)
	assert.Equal(t, cnts[len(cnts)-1], loaded)
	iter.Release()

}

func lexographicCountIndex(obj Object) ([]byte, error) {
	c, ok := obj.Value().(*CounterWithID)
	if !ok {
		return nil, errors.Wrapf(errors.ErrType, "%T", obj.Value())
	}
	res := make([]byte, 8)
	binary.BigEndian.PutUint64(res, uint64(c.Count))
	return res, nil
}

func TestSerialModelBucketIndexScanUnique(t *testing.T) {
	db := store.MemStore()

	b := NewSerialModelBucket("cnts", &CounterWithID{}, WithIndexSerial("counter", lexographicCountIndex, true))

	cnts := []CounterWithID{
		CounterWithID{Count: 1},
		CounterWithID{Count: 17},
		CounterWithID{Count: 93},
		CounterWithID{Count: 3},
		CounterWithID{Count: 8},
	}
	for i := range cnts {
		// make sure we point to value in array, so this ID gets set
		err := b.Put(db, &cnts[i])
		assert.Nil(t, err)
	}

	var loaded CounterWithID
	iter, err := b.IndexScan(db, "counter", nil, false)
	assert.Nil(t, err)

	err = iter.LoadNext(&loaded)
	assert.Nil(t, err)
	// should get lowest value
	assert.Equal(t, CounterWithID{ID: weavetest.SequenceID(1), Count: 1}, loaded)

	err = iter.LoadNext(&loaded)
	assert.Nil(t, err)
	// should get second-lowest value
	assert.Equal(t, CounterWithID{ID: weavetest.SequenceID(4), Count: 3}, loaded)

	iter.Release()

	// validate reverse also works
	iter, err = b.IndexScan(db, "counter", nil, true)
	assert.Nil(t, err)

	err = iter.LoadNext(&loaded)
	assert.Nil(t, err)
	// should get highest value
	assert.Equal(t, CounterWithID{ID: weavetest.SequenceID(3), Count: 93}, loaded)

	err = iter.LoadNext(&loaded)
	assert.Nil(t, err)
	// should get second-highest value
	assert.Equal(t, CounterWithID{ID: weavetest.SequenceID(2), Count: 17}, loaded)

	iter.Release()
}

func TestSerialModelBucketIndexScanMulti(t *testing.T) {
	db := store.MemStore()

	b := NewSerialModelBucket("cnts", &CounterWithID{}, WithIndexSerial("counter", lexographicCountIndex, false))

	cnts := []CounterWithID{
		CounterWithID{Count: 1},
		CounterWithID{Count: 17},
		CounterWithID{Count: 3},
		CounterWithID{Count: 8},
		CounterWithID{Count: 17},
		CounterWithID{Count: 3},
	}
	for i := range cnts {
		// make sure we point to value in array, so this ID gets set
		err := b.Put(db, &cnts[i])
		assert.Nil(t, err)
	}

	var loaded CounterWithID
	iter, err := b.IndexScan(db, "counter", nil, false)
	assert.Nil(t, err)
	err = iter.LoadNext(&loaded)
	assert.Nil(t, err)
	// should get lowest value
	assert.Equal(t, CounterWithID{ID: weavetest.SequenceID(1), Count: 1}, loaded)

	err = iter.LoadNext(&loaded)
	assert.Nil(t, err)
	// should get second-lowest value (3)
	assert.Equal(t, CounterWithID{ID: weavetest.SequenceID(3), Count: 3}, loaded)

	err = iter.LoadNext(&loaded)
	assert.Nil(t, err)
	// should get 3 a second time
	assert.Equal(t, CounterWithID{ID: weavetest.SequenceID(6), Count: 3}, loaded)

	iter.Release()

	// counting backwards
	iter, err = b.IndexScan(db, "counter", nil, true)
	assert.Nil(t, err)
	err = iter.LoadNext(&loaded)
	assert.Nil(t, err)
	// should get lowest value
	assert.Equal(t, CounterWithID{ID: weavetest.SequenceID(2), Count: 17}, loaded)

	err = iter.LoadNext(&loaded)
	assert.Nil(t, err)
	// should get second-lowest value (17)
	assert.Equal(t, CounterWithID{ID: weavetest.SequenceID(5), Count: 17}, loaded)

	err = iter.LoadNext(&loaded)
	assert.Nil(t, err)
	// should get third-lowest value (8)
	assert.Equal(t, CounterWithID{ID: weavetest.SequenceID(4), Count: 8}, loaded)

	iter.Release()

}

func TestSerialModelBucketByIndex(t *testing.T) {
	cases := map[string]struct {
		IndexName  string
		QueryKey   string
		DestFn     func() ModelSlicePtr
		WantErr    *errors.Error
		WantResPtr []*CounterWithID
		WantRes    []CounterWithID
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
			QueryKey:  "1",
			WantErr:   nil,
			WantResPtr: []*CounterWithID{
				{
					ID:    weavetest.SequenceID(1),
					Count: 1001,
				},
			},
			WantRes: []CounterWithID{
				{
					ID:    weavetest.SequenceID(1),
					Count: 1001,
				},
			},
		},
		"find two": {
			IndexName: "value",
			QueryKey:  "4",
			WantErr:   nil,
			WantResPtr: []*CounterWithID{
				{ID: weavetest.SequenceID(3), Count: 4001},
				{ID: weavetest.SequenceID(4), Count: 4002},
			},
			WantRes: []CounterWithID{
				{ID: weavetest.SequenceID(3), Count: 4001},
				{ID: weavetest.SequenceID(4), Count: 4002},
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

			indexByBigValue := func(obj Object) ([]byte, error) {
				c, ok := obj.Value().(*CounterWithID)
				if !ok {
					return nil, errors.Wrapf(errors.ErrType, "%T", obj.Value())
				}
				// Index by the value, ignoring anything below 1k.
				raw := strconv.FormatInt(c.Count/1000, 10)
				return []byte(raw), nil
			}
			b := NewSerialModelBucket("cnts", &CounterWithID{}, WithIndexSerial("value", indexByBigValue, false))

			err := b.Put(db, &CounterWithID{Count: 1001})
			assert.Nil(t, err)
			err = b.Put(db, &CounterWithID{Count: 2001})
			assert.Nil(t, err)
			err = b.Put(db, &CounterWithID{Count: 4001})
			assert.Nil(t, err)
			err = b.Put(db, &CounterWithID{Count: 4002})
			assert.Nil(t, err)

			var dest []CounterWithID
			err = b.ByIndex(db, tc.IndexName, []byte(tc.QueryKey), &dest)
			if !tc.WantErr.Is(err) {
				t.Fatalf("unexpected error: %s", err)
			}
			assert.Equal(t, tc.WantRes, dest)

			var destPtr []*CounterWithID
			err = b.ByIndex(db, tc.IndexName, []byte(tc.QueryKey), &destPtr)
			if !tc.WantErr.Is(err) {
				t.Fatalf("unexpected error: %s", err)
			}
			assert.Equal(t, tc.WantResPtr, destPtr)
		})
	}
}

// BadCounter implements Model but is different type than the model
type BadCounter struct {
	CounterWithID
	Random int
}

var _ Model = (*BadCounter)(nil)

func TestSerialModelBucketPutWrongModelType(t *testing.T) {
	db := store.MemStore()
	b := NewSerialModelBucket("cnts", &CounterWithID{})

	if err := b.Put(db, &BadCounter{CounterWithID: CounterWithID{Count: 5}, Random: 77}); !errors.ErrType.Is(err) {
		t.Fatalf("unexpected error when trying to store wrong model type value: %s", err)
	}
}

func TestSerialModelBucketOneWrongModelType(t *testing.T) {
	db := store.MemStore()
	b := NewSerialModelBucket("cnts", &CounterWithID{})

	err := b.Put(db, &CounterWithID{ID: []byte("counter"), Count: 1})
	assert.Nil(t, err)

	var bad BadCounter
	if err := b.One(db, []byte("counter"), &bad); !errors.ErrType.Is(err) {
		t.Fatalf("unexpected error when trying to get wrong model type value: %s", err)
	}
}

func TestSerialModelBucketByIndexWrongModelType(t *testing.T) {
	db := store.MemStore()
	indexer := func(o Object) ([]byte, error) {
		return []byte("x"), nil
	}
	b := NewSerialModelBucket("cnts", &CounterWithID{}, WithIndexSerial("x", indexer, false))

	err := b.Put(db, &CounterWithID{ID: []byte("counter"), Count: 1})
	assert.Nil(t, err)

	var bads []BadCounter
	if err := b.ByIndex(db, "x", []byte("x"), &bads); !errors.ErrType.Is(err) {
		t.Fatalf("unexpected error when trying to find wrong model type value: %s: %v", err, bads)
	}

	var badsPtr []*BadCounter
	if err := b.ByIndex(db, "x", []byte("x"), &badsPtr); !errors.ErrType.Is(err) {
		t.Fatalf("unexpected error when trying to find wrong model type value: %s: %v", err, badsPtr)
	}

	var badsPtrPtr []**BadCounter
	if err := b.ByIndex(db, "x", []byte("x"), &badsPtrPtr); !errors.ErrType.Is(err) {
		t.Fatalf("unexpected error when trying to find wrong model type value: %s: %v", err, badsPtrPtr)
	}
}

func TestSerialModelBucketHas(t *testing.T) {
	db := store.MemStore()
	b := NewSerialModelBucket("cnts", &CounterWithID{})

	err := b.Put(db, &CounterWithID{ID: []byte("counter"), Count: 1})
	assert.Nil(t, err)

	err = b.Has(db, []byte("counter"))
	assert.Nil(t, err)

	if err := b.Has(db, nil); !errors.ErrNotFound.Is(err) {
		t.Fatalf("a nil key must return ErrNotFound: %s", err)
	}

	if err := b.Has(db, []byte("does-not-exist")); !errors.ErrNotFound.Is(err) {
		t.Fatalf("a non exists entity must return ErrNotFound: %s", err)
	}
}
