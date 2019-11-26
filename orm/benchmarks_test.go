package orm

import (
"github.com/iov-one/weave"
"github.com/iov-one/weave/errors"
"github.com/iov-one/weave/store"
"testing"
)

func LoadAllBySecondaryIndex(db weave.ReadOnlyKVStore, domain string) (weave.Iterator, error) {
	const bucketPrefix = "prefix:"
	start := append([]byte(bucketPrefix + domain)) //, '*'-1)
	end := append([]byte(bucketPrefix+domain), '*'+1)
	return db.Iterator(start, end)
}

func itemKeys(it weave.Iterator, err error) ([][]byte, error) {
	if err != nil {
		return nil, err
	}
	defer it.Release()

	var keys [][]byte
	for {
		switch k, _, err := it.Next(); {
		case err == nil:
			keys = append(keys, k)
		case errors.ErrIteratorDone.Is(err):
			return keys, nil
		default:
			return keys, err
		}
	}
}

func dataKey(name, domain string) []byte {
	key := make([]byte, 0, len(name)+len(domain)+1)
	key = append(key, domain...)
	key = append(key, '*')
	key = append(key, name...)
	return key
}

func BenchmarkSecondaryIndex(b *testing.B) {
	benchmarks := []struct {
		name     string
		amount   int
		indexLen int
	}{
		{"index length 2 amount 1", 1, 2},
		{"index length 2 amount 10", 10, 2},
		{"index length 2 amount 100", 100, 2},
		{"index length 2 amount 1000", 1000, 2},
		{"index length 2 amount 10000", 10000, 2},
		{"index length 2 amount 50000", 50000, 2},

		{"index length 5 amount 1", 1, 5},
		{"index length 5 amount 10", 10, 5},
		{"index length 5 amount 100", 100, 5},
		{"index length 5 amount 1000", 1000, 5},
		{"index length 5 amount 10000", 10000, 5},
		{"index length 5 amount 50000", 50000, 5},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				db := store.MemStore()
				bucket := NewModelBucket("counter", &CounterWithID{})

				index := ""
				for i := 0; i < bm.indexLen; i ++ {
					index = index + "a"
				}
				sindex := ""
				for i := 0; i < bm.indexLen; i ++ {
					sindex = sindex + "a"
				}
				data := &CounterWithID{Index: index, Sindex: sindex}

				if _, err := bucket.Put(db, dataKey(index, sindex), data); err != nil {
					b.Error(err)
				}

				_, err := itemKeys(LoadAllBySecondaryIndex(db, sindex))
				if err != nil {
					b.Error(err)
				}
			}
		})
	}
}
