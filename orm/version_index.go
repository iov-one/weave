package orm

import (
	"bytes"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

type bucketGetter interface {
	Get(db weave.ReadOnlyKVStore, key []byte) (Object, error)
}
type VersionIndex struct {
	idx    Index
	bucket bucketGetter
}

func NewVersionIndex(name string, b bucketGetter) *VersionIndex {
	return &VersionIndex{
		idx:    NewIndex(name, nil, true),
		bucket: b,
	}
}
func (i VersionIndex) GetAt(db weave.ReadOnlyKVStore, index []byte) ([][]byte, error) {
	return i.idx.GetAt(db, index)
}

func (i VersionIndex) GetPrefix(db weave.ReadOnlyKVStore, prefix []byte) ([][]byte, error) {
	return i.idx.GetPrefix(db, prefix)
}

func (i VersionIndex) GetLike(db weave.ReadOnlyKVStore, pattern Object) ([][]byte, error) {
	return i.idx.GetLike(db, pattern)
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
func (i VersionIndex) Update(db weave.KVStore, prev Object, save Object) error {
	type s struct{ a, b bool }
	sw := s{prev == nil, save == nil}
	switch sw {
	case s{true, true}:
		return errors.Wrap(errors.ErrHuman, "update requires at least one non-nil object")
	case s{true, false}: // insert
		idRef, err := versionID(save)
		if err != nil {
			return err
		}
		objs, err := i.idx.GetAt(db, idRef.ID)
		switch {
		case err != nil:
			return err
		case len(objs) == 0: // no existing entry
			// then insert
			return i.idx.insert(db, idRef.ID, save.Key())
		case len(objs) != 1:
			return errors.Wrap(errors.ErrHuman, "multiple values stored for index key")
		}
		// otherwise replace existing entry
		obj, err := i.bucket.Get(db, objs[0])
		switch {
		case err != nil:
			return err
		case obj == nil:
			return errors.Wrap(errors.ErrNotFound, "index references a non existing obj")
		}
		latest, err := versionID(obj)
		switch {
		case err != nil:
			return err
		case !bytes.Equal(latest.ID, idRef.ID):
			return errors.Wrap(errors.ErrHuman, "non matching IDs")
		case latest.Version >= idRef.Version:
			return errors.Wrap(errors.ErrHuman, "version not greater than previous")
		}
		return i.move(db, idRef.ID, obj.Key(), save.Key())
	case s{false, false}: // update
		return errors.Wrap(errors.ErrHuman, "create new version instead of update")
	}
	return errors.Wrap(errors.ErrHuman, "you have violated the rules of boolean logic")
}

func (i VersionIndex) move(db weave.KVStore, id []byte, prevKey []byte, newKey []byte) error {
	if err := i.idx.remove(db, id, prevKey); err != nil {
		return err
	}
	return i.idx.insert(db, id, newKey)
}

func versionID(obj Object) (*VersionedIDRef, error) {
	if obj == nil {
		return nil, errors.Wrap(errors.ErrHuman, "cannot take index of nil")
	}
	var ref VersionedIDRef
	return &ref, ref.Unmarshal(obj.Key())
}
