package orm

import (
	"bytes"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

// SerialModelIterator over a domain of keys in ascending order. End is exclusive.
// Start must be less than end, or the Iterator is invalid.
// CONTRACT: No writes may happen within a domain while an iterator exists over it.
type SerialModelIterator interface {
	// LoadNext moves the iterator to the next sequential key in the database and
	// loads the current value at the given key into the passed destination.
	LoadNext(dest SerialModel) error

	// Release releases the Iterator.
	Release()
}

// idSerialModelIterator iterates over IDs
type idSerialModelIterator struct {
	// this is the raw KVStoreIterator
	iterator weave.Iterator
	// this is the bucketPrefix to strip from each key
	bucketPrefix []byte
}

var _ SerialModelIterator = (*idSerialModelIterator)(nil)

func (i *idSerialModelIterator) LoadNext(dest SerialModel) error {
	key, value, err := i.iterator.Next()
	if err != nil {
		return err
	}

	return load(key, value, i.bucketPrefix, dest)
}

func (i *idSerialModelIterator) Release() {
	i.iterator.Release()
}

// indexSerialModelIterator iterates over indexes
type indexSerialModelIterator struct {
	// this is the raw KVStoreIterator
	iterator weave.Iterator
	// this is the bucketPrefix to strip from each key
	bucketPrefix []byte
	unique       bool

	// there could be multiple indexes that point to the same
	// object so we cache keys.
	kv         weave.ReadOnlyKVStore
	cachedKeys [][]byte
}

var _ SerialModelIterator = (*indexSerialModelIterator)(nil)

// LoadNext loads next iterator value to dest
func (i *indexSerialModelIterator) LoadNext(dest SerialModel) error {
	key, err := i.getKey()
	if err != nil {
		return errors.Wrap(err, "cannot load next")
	}
	val, err := i.kv.Get(key)

	if err != nil {
		return errors.Wrap(err, "loading referenced key")
	}
	if val == nil {
		return errors.Wrapf(errors.ErrNotFound, "key: %X", key)
	}

	return load(key, val, i.bucketPrefix, dest)
}

func (i *indexSerialModelIterator) Release() {
	i.iterator.Release()
}

// getKey retrieves the key from cache if i.cacheKeys is not nil, otherwise loads next iterator key
func (i *indexSerialModelIterator) getKey() ([]byte, error) {
	var key []byte

	switch cachedKeysLen := len(i.cachedKeys); {
	case cachedKeysLen > 1:
		//gets the key from cache and remove first key from i.cacheKey
		key = i.dbKey(i.cachedKeys[0])
		i.cachedKeys = i.cachedKeys[1:]
	case cachedKeysLen == 1:
		//gets the key from cache and sets i.cachedKeys as nil
		key = i.dbKey(i.cachedKeys[0])
		i.cachedKeys = nil
	default:
		//retrievesthe key and value from iterator
		_, value, err := i.iterator.Next()
		if err != nil {
			return nil, err
		}

		keys, err := i.getRefs(value, i.unique)
		if err != nil {
			return nil, errors.Wrap(err, "parsing index refs")
		}
		if len(keys) != 1 {
			i.cachedKeys = keys[1:]
		}
		key = i.dbKey(keys[0])
	}

	return key, nil
}

// get refs takes a value stored in an index and parse it into a slice of
// db keys
func (i *indexSerialModelIterator) getRefs(val []byte, unique bool) ([][]byte, error) {
	if val == nil {
		return nil, nil
	}
	if unique {
		return [][]byte{val}, nil
	}
	var data = new(MultiRef)
	err := data.Unmarshal(val)
	if err != nil {
		return nil, err
	}
	return data.GetRefs(), nil
}

func (i *indexSerialModelIterator) dbKey(key []byte) []byte {
	return append(i.bucketPrefix, key...)
}

func load(key, value, bucketPrefix []byte, dest SerialModel) error {
	// since we use raw kvstore here, not Bucket, we must remove the bucket prefix manually
	if !bytes.HasPrefix(key, bucketPrefix) {
		return errors.Wrapf(errors.ErrDatabase, "key with unexpected prefix: %X", key)
	}
	key = key[len(bucketPrefix):]

	if err := dest.Unmarshal(value); err != nil {
		return errors.Wrapf(err, "unmarshaling into %T", dest)
	}
	if err := dest.SetID(key); err != nil {
		return errors.Wrap(err, "setting ID")
	}
	return nil
}
