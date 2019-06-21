package orm

import (
	"bytes"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

var indPrefix = []byte("_i.")

// Indexer calculates the secondary index key for a given object
type Indexer func(Object) ([]byte, error)

// MultiKeyIndexer calculates the secondary index keys for a given object
type MultiKeyIndexer func(Object) ([][]byte, error)

// Index represents a secondary index on some data.
// It is indexed by an arbitrary key returned by Indexer.
// The value is one primary key (unique),
// Or an array of primary keys (!unique).
type Index struct {
	name   string
	id     []byte
	unique bool
	index  MultiKeyIndexer
	refKey func([]byte) []byte
}

var _ weave.QueryHandler = Index{}

// NewIndex constructs an index with single key Indexer.
// Indexer calculates the index for an object
// unique enforces a unique constraint on the index
// refKey calculates the absolute dbkey for a ref
func NewIndex(name string, indexer Indexer, unique bool,
	refKey func([]byte) []byte) Index {
	return NewMultiKeyIndex(name, asMultiKeyIndexer(indexer), unique, refKey)
}

// NewMultiKeyIndex constructs an index with multi key indexer.
// Indexer calculates the index for an object
// unique enforces a unique constraint on the index
// refKey calculates the absolute dbkey for a ref
func NewMultiKeyIndex(name string, indexer MultiKeyIndexer, unique bool,
	refKey func([]byte) []byte) Index {
	// TODO: index name must be [a-z_]
	return Index{
		name:   name,
		id:     append(indPrefix, []byte(name+":")...),
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

// IndexKey is the full key we store in the db, including prefix
// We copy into a new array rather than use append, as we don't
// want consecutive calls to overwrite the same byte array.
func (i Index) IndexKey(key []byte) []byte {
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
func (i Index) Update(db weave.KVStore, prev Object, save Object) error {
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

// GetLike calculates the index for the given pattern, and
// returns a list of all pk that match (may be nil when empty), or an error
func (i Index) GetLike(db weave.ReadOnlyKVStore, pattern Object) ([][]byte, error) {
	indexes, err := i.index(pattern)
	if err != nil {
		return nil, err
	}
	var r [][]byte
	for _, index := range indexes {
		pks, err := i.GetAt(db, index)
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

// GetAt returns a list of all pk at that index (may be empty), or an error
func (i Index) GetAt(db weave.ReadOnlyKVStore, index []byte) ([][]byte, error) {
	key := i.IndexKey(index)
	val, err := db.Get(key)
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, nil
	}
	if i.unique {
		return [][]byte{val}, nil
	}
	var data = new(MultiRef)
	err = data.Unmarshal(val)
	if err != nil {
		return nil, err
	}
	return data.GetRefs(), nil
}

// GetPrefix returns all references that have an index that
// begins with a given prefix
func (i Index) GetPrefix(db weave.ReadOnlyKVStore, prefix []byte) ([][]byte, error) {
	dbPrefix := i.IndexKey(prefix)
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
func (i Index) Query(db weave.ReadOnlyKVStore, mod string,
	data []byte) ([]weave.Model, error) {

	switch mod {
	case weave.KeyQueryMod:
		refs, err := i.GetAt(db, data)
		if err != nil {
			return nil, err
		}
		return i.loadRefs(db, refs)
	case weave.PrefixQueryMod:
		refs, err := i.GetPrefix(db, data)
		if err != nil {
			return nil, err
		}
		return i.loadRefs(db, refs)
	default:
		return nil, errors.Wrap(errors.ErrHuman, "not implemented: "+mod)
	}
}

func (i Index) loadRefs(db weave.ReadOnlyKVStore,
	refs [][]byte) ([]weave.Model, error) {

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

func (i Index) move(db weave.KVStore, prev Object, save Object) error {
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
			k := i.IndexKey(newKey)
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

func (i Index) remove(db weave.KVStore, index []byte, pk []byte) error {
	// don't deal with empty keys
	if len(index) == 0 {
		return nil
	}

	key := i.IndexKey(index)
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

func (i Index) insert(db weave.KVStore, index []byte, pk []byte) error {
	// don't deal with empty keys
	if len(index) == 0 {
		return nil
	}

	key := i.IndexKey(index)
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
