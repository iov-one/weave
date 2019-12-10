package orm

import (
	"bytes"
	math "math"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

type Index interface {
	// Name returns the name of this index.
	Name() string

	// Update updates the index. It should be called when any of the bucket
	// entities has changed in the store.
	//
	// prev == nil means insert
	// save == nil means delete
	// both == nil is error
	// if both != nil and prev.Key() != save.Key() this is an error
	Update(db weave.KVStore, prev Object, save Object) error

	// Keys returns an iteator that returns all entity keys that were
	// indexed under given value.
	//
	// Values of returned iterator are always nil to optimize for a lazy
	// loading flows and avoid loading into memory values from the database
	// when they might not be needed.
	Keys(db weave.ReadOnlyKVStore, value []byte) weave.Iterator

	// Query handles queries from the QueryRouter.
	Query(db weave.ReadOnlyKVStore, mod string, data []byte) ([]weave.Model, error)
}

const compactIdxPrefix = "_i."

// Indexer calculates the secondary index key for a given object
type Indexer func(Object) ([]byte, error)

// MultiKeyIndexer calculates the secondary index keys for a given object
type MultiKeyIndexer func(Object) ([][]byte, error)

// compactIndex is an index implementation that stores all indexed entities as
// a set, serialized and stored under single key. This implmentation should be
// used only for small sized index collection. Use nativeIndex for big indexes.
//
// compactIndex represents a secondary index on some data.
// It is indexed by an arbitrary key returned by Indexer.
// The value is one primary key (unique),
// Or an array of primary keys (!unique).
type compactIndex struct {
	name   string
	id     []byte
	unique bool
	index  MultiKeyIndexer
	refKey func([]byte) []byte
}

var _ weave.QueryHandler = compactIndex{}

// NewMultiKeyIndex constructs an index with multi key indexer.
// Indexer calculates the index for an object
// unique enforces a unique constraint on the index
// refKey calculates the absolute dbkey for a ref
func NewMultiKeyIndex(name string, indexer MultiKeyIndexer, unique bool, refKey func([]byte) []byte) Index {
	// TODO: index name must be [a-z_]
	return compactIndex{
		name:   name,
		id:     append([]byte(compactIdxPrefix), []byte(name+":")...),
		index:  indexer,
		unique: unique,
		refKey: refKey,
	}
}

func asMultiKeyIndexer(indexer Indexer) MultiKeyIndexer {
	return func(obj Object) ([][]byte, error) {
		key, err := indexer(obj)
		switch {
		case err != nil:
			return nil, err
		case key == nil:
			return nil, nil
		}
		return [][]byte{key}, nil
	}
}

func (i compactIndex) Name() string {
	return i.name
}

// indexKey is the full key we store in the db, including prefix
// We copy into a new array rather than use append, as we don't
// want consecutive calls to overwrite the same byte array.
func (i compactIndex) indexKey(key []byte) []byte {
	l := len(i.id)
	out := make([]byte, l+len(key))
	copy(out, i.id)
	copy(out[l:], key)
	return out
}

// Update handles updating the reference to the object in
// the secondary index.
//
// prev == nil means insert
// save == nil means delete
// both == nil is error
// if both != nil and prev.Key() != save.Key() this is an error
//
// Otherwise, it will check indexer(prev) and indexer(save)
// and make sure the key is now stored in the right location
func (i compactIndex) Update(db weave.KVStore, prev Object, save Object) error {
	type s struct{ a, b bool }
	sw := s{prev == nil, save == nil}
	switch sw {
	case s{true, true}:
		return errors.Wrap(errors.ErrHuman, "update requires at least one non-nil object")
	case s{true, false}:
		keys, err := i.index(save)
		if err != nil {
			return err
		}
		for _, key := range keys {
			if err := i.insert(db, key, save.Key()); err != nil {
				return err
			}
		}
		return nil
	case s{false, true}:
		keys, err := i.index(prev)
		if err != nil {
			return err
		}
		for _, key := range keys {
			if err := i.remove(db, key, prev.Key()); err != nil {
				return err
			}
		}
		return nil
	case s{false, false}:
		return i.move(db, prev, save)
	}
	return errors.Wrap(errors.ErrHuman, "you have violated the rules of boolean logic")
}

// Like calculates the index for the given pattern, and
// returns a list of all pk that match (may be nil when empty), or an error
func (i compactIndex) Like(db weave.ReadOnlyKVStore, pattern Object) ([][]byte, error) {
	indexes, err := i.index(pattern)
	if err != nil {
		return nil, err
	}
	var r [][]byte
	for _, index := range indexes {
		pks, err := consumeIteratorKeys(i.Keys(db, index))
		if err != nil {
			return nil, err
		}
		if i.unique {
			return pks, nil
		}
		r = append(r, pks...)
	}
	return deduplicate(r), nil
}

func deduplicate(s [][]byte) [][]byte {
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if bytes.Equal(s[i], s[j]) {
				s = append(s[0:j], s[j+1:]...)
			}
		}
	}
	return s
}

// Keys returns a list of all entity keys that were indexed under given value.
func (i compactIndex) Keys(db weave.ReadOnlyKVStore, index []byte) weave.Iterator {
	key := i.indexKey(index)
	val, err := db.Get(key)
	if err != nil {
		return &failedIterator{err: err}
	}
	if val == nil {
		return &failedIterator{err: errors.ErrIteratorDone}
	}
	if i.unique {
		return &keysIterator{keys: [][]byte{val}}
	}

	var data MultiRef
	if err := data.Unmarshal(val); err != nil {
		return &failedIterator{err: err}
	}
	return &keysIterator{keys: data.GetRefs()}
}

type failedIterator struct {
	err error
}

var _ weave.Iterator = (*failedIterator)(nil)

func (it *failedIterator) Next() ([]byte, []byte, error) {
	return nil, nil, it.err
}

func (failedIterator) Release() {}

type keysIterator struct {
	keys [][]byte
}

var _ weave.Iterator = (*keysIterator)(nil)

func (it *keysIterator) Next() ([]byte, []byte, error) {
	if len(it.keys) == 0 {
		return nil, nil, errors.ErrIteratorDone
	}
	key := it.keys[0]
	it.keys = it.keys[1:]
	return key, nil, nil
}

func (keysIterator) Release() {}

// consumeIteratorKeys returns a list of all keys that given iterator returns.
// This function should be used only for iterators when the result size is
// known to be small as all results are kept in memory.
// This function releases the iterator.
func consumeIteratorKeys(it weave.Iterator) ([][]byte, error) {
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

// getPrefix returns all references that have an index that
// begins with a given prefix
func (i compactIndex) getPrefix(db weave.ReadOnlyKVStore, prefix []byte) ([][]byte, error) {
	dbPrefix := i.indexKey(prefix)
	itr, err := db.Iterator(prefixRange(dbPrefix))
	if err != nil {
		return nil, err
	}
	defer itr.Release()

	var data [][]byte
	_, value, err := itr.Next()
	for err == nil {
		if i.unique {
			data = append(data, value)
		} else {
			tmp := new(MultiRef)
			err := tmp.Unmarshal(value)
			if err != nil {
				return nil, err
			}
			data = append(data, tmp.Refs...)
		}
		_, value, err = itr.Next()
	}
	if !errors.ErrIteratorDone.Is(err) {
		return nil, err
	}
	return data, nil
}

// Query handles queries from the QueryRouter
func (i compactIndex) Query(db weave.ReadOnlyKVStore, mod string, data []byte) ([]weave.Model, error) {
	switch mod {
	case weave.KeyQueryMod:
		refs, err := consumeIteratorKeys(i.Keys(db, data))
		if err != nil {
			return nil, err
		}
		return i.loadRefs(db, refs)
	case weave.PrefixQueryMod:
		refs, err := i.getPrefix(db, data)
		if err != nil {
			return nil, err
		}
		return i.loadRefs(db, refs)
	default:
		return nil, errors.Wrap(errors.ErrHuman, "not implemented: "+mod)
	}
}

func (i compactIndex) loadRefs(db weave.ReadOnlyKVStore, refs [][]byte) ([]weave.Model, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	res := make([]weave.Model, len(refs))
	for j, ref := range refs {
		key := i.refKey(ref)
		value, err := db.Get(key)
		if err != nil {
			return nil, err
		}
		res[j] = weave.Model{
			Key:   key,
			Value: value,
		}
	}
	return res, nil
}

func (i compactIndex) move(db weave.KVStore, prev Object, save Object) error {
	// if the primary key is not equal, we have a problem
	if !bytes.Equal(prev.Key(), save.Key()) {
		return errors.Wrap(errors.ErrImmutable, "cannot modify the primary key of an object")
	}

	oldKeys, err := i.index(prev)
	if err != nil {
		return err
	}
	newKeys, err := i.index(save)
	if err != nil {
		return err
	}
	keysToAdd := subtract(newKeys, oldKeys)
	keysToRemove := subtract(oldKeys, newKeys)

	// check unique constraints first
	for _, newKey := range keysToAdd {
		if i.unique {
			k := i.indexKey(newKey)
			val, err := db.Get(k)
			if err != nil {
				return err
			}
			if val != nil {
				return errors.Wrap(errors.ErrDuplicate, i.name)
			}
		}
	}

	// remove unused keys
	for _, oldKey := range keysToRemove {
		if err = i.remove(db, oldKey, prev.Key()); err != nil {
			return err
		}
	}

	// add new keys
	for _, newKey := range keysToAdd {
		if err = i.insert(db, newKey, prev.Key()); err != nil {
			return err
		}
	}
	return nil
}

// subtract returns all elements of minuend that are not in subtrahend.
func subtract(minuend [][]byte, subtrahend [][]byte) [][]byte {
	if minuend == nil {
		return nil
	}
	r := make([][]byte, 0)
OUTER:
	for _, m := range minuend {
		for _, s := range subtrahend {
			if bytes.Equal(m, s) {
				continue OUTER
			}
		}
		r = append(r, m)
	}
	return r
}

func (i compactIndex) remove(db weave.KVStore, index []byte, pk []byte) error {
	// don't deal with empty keys
	if len(index) == 0 {
		return nil
	}

	key := i.indexKey(index)
	cur, err := db.Get(key)
	if err != nil {
		return err
	}
	if cur == nil {
		return errors.Wrap(errors.ErrNotFound, "cannot remove index from nothing")
	}
	if i.unique {
		// if something else was here, don't delete
		if !bytes.Equal(cur, pk) {
			return errors.Wrap(errors.ErrNotFound, "cannot remove index from invalid object")
		}
		return db.Delete(key)
	}

	// otherwise, remove one from a list....
	var data = new(MultiRef)
	err = data.Unmarshal(cur)
	if err != nil {
		return err
	}
	err = data.Remove(pk)
	if err != nil {
		return err
	}
	// nothing left, delete this key
	if data.Size() == 0 {
		return db.Delete(key)
	}
	// other left, just update state
	save, err := data.Marshal()
	if err != nil {
		return err
	}

	return db.Set(key, save)
}

func (i compactIndex) insert(db weave.KVStore, index []byte, pk []byte) error {
	// don't deal with empty keys
	if len(index) == 0 {
		return nil
	}

	key := i.indexKey(index)
	cur, err := db.Get(key)
	if err != nil {
		return err
	}

	if i.unique {
		if cur != nil {
			return errors.Wrap(errors.ErrDuplicate, i.name)
		}

		return db.Set(key, pk)
	}

	// otherwise, add one to a list....
	var data = new(MultiRef)
	if cur != nil {
		err := data.Unmarshal(cur)
		if err != nil {
			return err
		}
	}
	err = data.Add(pk)
	if err != nil {
		return err
	}

	// other left, just update state
	save, err := data.Marshal()
	if err != nil {
		return err
	}

	return db.Set(key, save)
}

const nativeIdxPrefix = "_x."

// NewNativeIndex returns an index implementation that is using a database
// native storage and query in order to maintain and provide access to an
// index.
func NewNativeIndex(name string, indexer MultiKeyIndexer) Index {
	return &nativeIndex{
		name:    name,
		indexer: indexer,
	}
}

// nativeIndex is an index implementation that is using a database native
// storage and query in order to maintain and provide access to an index.
type nativeIndex struct {
	name    string
	indexer MultiKeyIndexer
}

func (ix *nativeIndex) Name() string {
	return ix.name
}

// Update updates the index. It should be called when any of the bucket
// entities has changed in the store.
//
// prev == nil means insert
// next == nil means delete
// both == nil is error
// if both != nil and prev.Key() != next.Key() this is an error
func (ix *nativeIndex) Update(db weave.KVStore, prev Object, next Object) error {
	if next == nil && prev == nil {
		return errors.Wrap(errors.ErrInput, "update requires at least one non-nil object")
	}
	if next != nil && prev != nil {
		if !bytes.Equal(next.Key(), prev.Key()) {
			return errors.Wrap(errors.ErrState, "previous key is not the same as the new one")
		}
	}

	// Delete.
	if prev != nil {
		values, err := ix.indexer(prev)
		if err != nil {
			return errors.Wrap(err, "indexer")
		}
		for _, v := range values {
			idxKey, err := packNativeIdxKey([][]byte{[]byte(ix.name), v, prev.Key()})
			if err != nil {
				return errors.Wrap(err, "build index key")
			}
			if err := db.Delete(idxKey); err != nil {
				return errors.Wrap(err, "db delete")
			}
		}
	}

	// Insert.
	if next != nil {
		values, err := ix.indexer(next)
		if err != nil {
			return errors.Wrap(err, "indexer")
		}
		for _, v := range values {
			idxKey, err := packNativeIdxKey([][]byte{[]byte(ix.name), v, next.Key()})
			if err != nil {
				return errors.Wrap(err, "build index key")
			}
			if err := db.Set(idxKey, nil); err != nil {
				return errors.Wrap(err, "db set")
			}
		}
	}

	return nil
}

func (ix *nativeIndex) Keys(db weave.ReadOnlyKVStore, value []byte) weave.Iterator {
	lookupKey, err := packNativeIdxKey([][]byte{[]byte(ix.name), value})
	if err != nil {
		return &failedIterator{err: errors.Wrap(err, "build index key")}
	}

	// Index key are built is a specific way, that allow using the native
	// database key iteration in order to find all indexed entries. Index
	// key is in format:
	//    <prefix>#<index name>#<value>#<entity id>
	// where # is a serialization specific data, irrelevant for the
	// algorithm.
	// To iterate over all values matching given index, iterate over all
	// keys between:
	//    <prefix>#<index name>#<value> and <prefix>#<index name>#<value>{255}
	//
	// Parse matching keys and return the last part of it, being the
	// indexed entity.
	// Value 255 is reserved to make sure no indexed key is matching it
	// (see packNativeIdxKey function).

	start := lookupKey
	end := make([]byte, len(lookupKey)+1)
	copy(end, lookupKey)
	// MaxUint8 is not used by serializer so we can use it as the maximum
	// value guard.
	end[len(end)-1] = math.MaxUint8

	it, err := db.Iterator(start, end)
	if err != nil {
		return &failedIterator{err: err}
	}

	return &nativeIndexIterator{dbit: it}
}

// nativeIndexIterator wraps a database iterator and parse results to provide
// indexed entities keys. It provides an interface that returns only the
// relevant data, hiding from the user native index implementation details.
type nativeIndexIterator struct {
	dbit weave.Iterator
}

func (it *nativeIndexIterator) Release() {
	it.dbit.Release()
}

func (it *nativeIndexIterator) Next() ([]byte, []byte, error) {
	key, _, err := it.dbit.Next()
	if err != nil {
		return key, nil, err
	}
	chunks, err := unpackNativeIdxKey(key)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unpack native index key")
	}
	return chunks[len(chunks)-1], nil, nil
}

func (ix *nativeIndex) Query(db weave.ReadOnlyKVStore, mod string, data []byte) ([]weave.Model, error) {
	// Query method is required by the Index interface. This implementation
	// can build huge indexes. While compactIndex can always return all
	// results in a single response, nativeIndex should never do this.
	// Native index implementation expects huge result sets and returning
	// them could enable DDOS attack.
	return nil, errors.Wrap(errors.ErrHuman, "not implemented: "+mod)
}

func packNativeIdxKey(chunks [][]byte) ([]byte, error) {
	var size int
	for _, b := range chunks {
		size += len(b) + 1
	}
	// First bytes are prefix information.
	res := make([]byte, 0, size+len(nativeIdxPrefix))
	res = append(res, nativeIdxPrefix...)

	for _, b := range chunks {
		// MaxUint8 is reserved for the search purpose. MaxUint8 - 1 is
		// the greatest allowed length.
		if len(b) > math.MaxUint8-1 {
			return nil, errors.Wrapf(errors.ErrInput, "no chunk can be bigger than %d bytes", math.MaxUint8-1)
		}
		res = append(res, uint8(len(b)))
		res = append(res, b...)
	}
	return res, nil
}

func unpackNativeIdxKey(b []byte) ([][]byte, error) {
	if len(b) < len(nativeIdxPrefix) {
		return nil, errors.Wrap(errors.ErrInput, "not a native index key")
	}
	if !bytes.Equal(b[:len(nativeIdxPrefix)], []byte(nativeIdxPrefix)) {
		return nil, errors.Wrap(errors.ErrInput, "not a native index key")
	}
	b = b[len(nativeIdxPrefix):]
	res := make([][]byte, 0, 6)
	for len(b) > 0 {
		size := uint8(b[0])
		if len(b) < 1+int(size) {
			return nil, errors.Wrap(errors.ErrInput, "malformed offset")
		}
		res = append(res, b[1:1+size])
		b = b[1+size:]
	}
	return res, nil
}
