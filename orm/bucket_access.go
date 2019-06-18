package orm

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

type visitableBucket interface {
	visit(func(rawBucket BaseBucket))
}

func Parse(b visitableBucket, key, value []byte) (obj Object, err error) {
	b.visit(func(rawBucket BaseBucket) {
		obj, err = rawBucket.parse(key, value)
	})
	return
}

func DBKey(b visitableBucket, key []byte) (result []byte) {
	b.visit(func(rawBucket BaseBucket) {
		result = rawBucket.dbKey(key)
	})
	return
}

func Query(b visitableBucket, db weave.ReadOnlyKVStore, mod string, data []byte) (m []weave.Model, err error) {
	b.visit(func(rawBucket BaseBucket) {
		m, err = rawBucket.Query(db, mod, data)
	})
	return
}

func WithQueryAdaptor(b visitableBucket) weave.QueryHandler {
	return weave.QueryHandlerFunc(func(db weave.ReadOnlyKVStore, mod string, data []byte) ([]weave.Model, error) {
		switch mod {
		case weave.KeyQueryMod:
			key := DBKey(b, data)
			value, err := db.Get(key)
			if err != nil {
				return nil, err
			}
			if value == nil {
				return nil, nil
			}
			res := []weave.Model{{Key: key, Value: value}}
			return res, nil
		case weave.PrefixQueryMod:
			prefix := DBKey(b, data)
			return queryPrefix(db, prefix)
		default:
			return nil, errors.Wrapf(errors.ErrInput, "unknown mod: %s", mod)
		}
	})
}
