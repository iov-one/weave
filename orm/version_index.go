package orm

import (
	"bytes"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

type VersionIndex struct {
	idx Index
}

func NewVersionIndex(name string) *VersionIndex {
	return &VersionIndex{
		idx: NewIndex(name, nil, true),
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
	case s{true, false}: // delete
		idRef, err := versionID(save)
		if err != nil {
			return err
		}
		return i.idx.insert(db, idRef.ID, save.Key())
	case s{false, true}: // insert
		idRef, err := versionID(prev)
		if err != nil {
			return err
		}
		if v := db.Get(idRef.ID); v != nil {
			return errors.ErrDuplicate
		}
		// TODO: revisit this: store empty ref key to mark deleted or drop value but that is not the same as never existed before
		return i.move(db, idRef.ID, prev.Key(), []byte{})
	case s{false, false}: // update
		prevVersionID, err := versionID(prev)
		if err != nil {
			return err
		}
		saveVersionID, err := versionID(save)
		switch {
		case err != nil:
			return err
		case !bytes.Equal(prevVersionID.ID, saveVersionID.ID):
			return errors.Wrap(errors.ErrHuman, "non matching IDs")
		case prevVersionID.Version >= saveVersionID.Version:
			return errors.Wrap(errors.ErrHuman, "version not greater than previous")
		}
		return i.move(db, prevVersionID.ID, prev.Key(), save.Key())
	}
	return errors.Wrap(errors.ErrHuman, "you have violated the rules of boolean logic")
}

func (i VersionIndex) move(db weave.KVStore, id []byte, prevKey []byte, newKey []byte) error {
	if v := db.Get(id); v != nil {
		return errors.ErrDuplicate
	}
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
