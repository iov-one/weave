package orm

import (
	"github.com/iov-one/weave"
)

type EmbeddedBucket interface {
	parent() EmbeddedBucket
}
type VisitableBucket interface {
	visit(func(rawBucket BaseBucket))
}

func Parse(b VisitableBucket, key, value []byte) (obj Object, err error) {
	b.visit(func(rawBucket BaseBucket) {
		obj, err = rawBucket.parse(key, value)
	})
	return
}

func DBKey(b VisitableBucket, key []byte) (result []byte) {
	b.visit(func(rawBucket BaseBucket) {
		result = rawBucket.dbKey(key)
	})
	return
}

func Register(b VisitableBucket, name string, r weave.QueryRouter) {
	b.visit(func(rawBucket BaseBucket) {
		rawBucket.Register(name, r)
	})
}

// Query the data in the bucket
func Query(b VisitableBucket, db weave.ReadOnlyKVStore, mod string, data []byte) (m []weave.Model, err error) {
	b.visit(func(rawBucket BaseBucket) {
		m, err = rawBucket.Query(db, mod, data)
	})
	return
}
