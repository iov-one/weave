package orm

import (
	"bytes"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

type VersioningBucket struct {
	IDGenBucket
}

type versioned interface {
	GetVersion() uint32
}

type versionedData interface {
	CloneableData
	versioned
	SetVersion(uint32)
}

func WithVersioning(b IDGenBucket) VersioningBucket {
	return VersioningBucket{b}
}

// Return
//  - ErrNotFound when not found
//  - ErrDeleted when deleted
// Object won't be nil in success case
func (b VersioningBucket) GetLatestVersion(db weave.ReadOnlyKVStore, id []byte) (VersionedIDRef, Object, error) {
	idWithoutVersion := VersionedIDRef{ID: id}
	prefix, err := idWithoutVersion.Marshal()
	if err != nil {
		return idWithoutVersion, nil, errors.Wrap(err, "failed to marshal versioned ID ref")
	}
	dbKeyLength := len(b.DBKey(prefix)) - len(prefix)
	matches, err := b.Query(db, weave.PrefixQueryMod, prefix)
	if err != nil {
		return idWithoutVersion, nil, errors.Wrap(err, "prefix query")
	}
	// find highest version for that ID
	var highestVersion VersionedIDRef
	var found weave.Model
	for _, m := range matches {
		var vID VersionedIDRef
		idData := m.Key[dbKeyLength:]
		if err := vID.Unmarshal(idData); err != nil {
			return idWithoutVersion, nil, errors.Wrap(err, "wrong key type")
		}
		if vID.Version > highestVersion.Version {
			highestVersion = vID
			found = m
		}
	}
	if len(found.Key) == 0 {
		return idWithoutVersion, nil, errors.Wrap(errors.ErrNotFound, "unknown id")
	}
	if tombstone.Equal(found.Value) {
		return idWithoutVersion, nil, errors.ErrDeleted
	}
	obj, err := b.Parse(found.Key, found.Value)
	if err != nil {
		return idWithoutVersion, nil, err
	}
	return highestVersion, obj, err
}

// Return
//  - ErrNotFound when not found
//  - ErrDeleted when deleted
// Object won't be nil in success case
func (b VersioningBucket) Get(db weave.ReadOnlyKVStore, key []byte) (Object, error) {
	bz, err := db.Get(b.DBKey(key))
	switch {
	case err != nil:
		return nil, err
	case bz == nil:
		return nil, errors.ErrNotFound
	case tombstone.Equal(bz):
		return nil, errors.ErrDeleted
	}
	return b.Parse(key, bz)
}

func (b VersioningBucket) GetVersion(db weave.ReadOnlyKVStore, ref VersionedIDRef) (Object, error) {
	key, err := ref.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal version id key")
	}
	return b.Get(db, key)
}

// Create assigns an ID and initial version number to given object instance and returns it as an persisted orm
// Object.
func (b VersioningBucket) Create(db weave.KVStore, data versionedData) (*VersionedIDRef, error) {
	if data.GetVersion() != 0 {
		return nil, errors.Wrap(errors.ErrInvalidInput, "version is set on create")
	}
	data.SetVersion(1)
	newID, err := b.idGen.NextVal(db, data)
	if err != nil {
		return nil, err
	}
	idRef := VersionedIDRef{ID: newID, Version: data.GetVersion()}
	return b.safeUpdate(db, idRef, data)
}

// Deprecated: Save will always return an error. Use Create or Update instead.
func (b VersioningBucket) Save(db weave.KVStore, model Object) error {
	return errors.Wrap(errors.ErrHuman, "raw save not supported")
}

// Update persists the given data object with a new derived version key in the storage.
func (b VersioningBucket) Update(db weave.KVStore, currentKey VersionedIDRef, data versionedData) (*VersionedIDRef, error) {
	if data.GetVersion() != currentKey.Version {
		return nil, errors.Wrap(errors.ErrInvalidState, "versions not matching")
	}
	// prevent gaps
	switch exists, err := b.Exists(db, currentKey); {
	case err != nil:
		return nil, err
	case !exists:
		return nil, errors.Wrap(errors.ErrNotFound, "current key")
	}
	newVersionKey, err := currentKey.NextVersion()
	if err != nil {
		return nil, err
	}

	// prevent overwrites
	switch existingObj, err := b.Exists(db, newVersionKey); {
	case err != nil:
		return nil, err
	case existingObj:
		return nil, errors.Wrap(errors.ErrDuplicate, "exists already")
	}
	data.SetVersion(newVersionKey.Version)
	return b.safeUpdate(db, newVersionKey, data)
}

func (b VersioningBucket) safeUpdate(db weave.KVStore, newVersionKey VersionedIDRef, data CloneableData) (*VersionedIDRef, error) {
	key, err := newVersionKey.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshall versioned id ref")
	}
	// store new version
	return &newVersionKey, b.Bucket.Save(db, NewSimpleObj(key, data))
}

func (b VersioningBucket) Exists(db weave.KVStore, idRef VersionedIDRef) (bool, error) {
	_, err := b.GetVersion(db, idRef)
	switch {
	case err == nil:
		return true, nil
	case errors.ErrNotFound.Is(err):
		return false, nil
	default:
		return false, errors.Wrap(err, "failed to load object")
	}
}

// Delete stores an nil value for the new highest version. It will return this key and nil on success.
// It return ErrNotFound when id does not exist
func (b VersioningBucket) Delete(db weave.KVStore, id []byte) (*VersionedIDRef, error) {
	latestKey, _, err := b.GetLatestVersion(db, id)
	if err != nil {
		return nil, err
	}
	newVersionKey, err := latestKey.NextVersion()
	if err != nil {
		return nil, err
	}
	return b.safeUpdate(db, newVersionKey, tombstone)
}

var tombstone = marker([]byte{})

type marker []byte

func (m marker) Validate() error {
	return nil
}

func (m marker) Marshal() ([]byte, error) {
	return m, nil
}

func (marker) Unmarshal([]byte) error {
	return nil
}

func (marker) Copy() CloneableData {
	return marker{}
}

func (m marker) Equal(o []byte) bool {
	return bytes.Equal(m, o)
}
