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

// tombstone is a null value object used internally
var tombstone = marker([]byte{})

// WithVersioning add versioning functionality to the underlying bucket. This means objects can not be overwritten
// anymore via Save function which must not be used with this type.
// Instead Create and Update methods are provided to support a history of object versions.
func WithVersioning(b IDGenBucket) VersioningBucket {
	return VersioningBucket{b}
}

// GetLatestVersion finds the latest version for the given id and returns the VersionedIDRef and loaded object.
// Unlike the classic Get function it returns:
//  - ErrNotFound when not found
//  - ErrDeleted when deleted
// Object won't be nil in success case
func (b VersioningBucket) GetLatestVersion(db weave.ReadOnlyKVStore, id []byte) (*VersionedIDRef, Object, error) {
	idWithoutVersion := VersionedIDRef{ID: id}
	prefix, err := idWithoutVersion.Marshal()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to marshal versioned ID ref")
	}
	dbKeyLength := len(b.DBKey(prefix)) - len(prefix)
	matches, err := b.Query(db, weave.PrefixQueryMod, prefix)
	if err != nil {
		return nil, nil, errors.Wrap(err, "prefix query")
	}
	// find highest version for that ID
	var highestVersion VersionedIDRef
	var found weave.Model
	for _, m := range matches {
		var vID VersionedIDRef
		idData := m.Key[dbKeyLength:]
		if err := vID.Unmarshal(idData); err != nil {
			return nil, nil, errors.Wrap(err, "wrong key type")
		}
		if vID.Version > highestVersion.Version {
			highestVersion = vID
			found = m
		}
	}
	if len(found.Key) == 0 {
		return nil, nil, errors.Wrap(errors.ErrNotFound, "unknown id")
	}
	if tombstone.Equal(found.Value) {
		return nil, nil, errors.ErrDeleted
	}
	obj, err := b.Parse(found.Key, found.Value)
	if err != nil {
		return nil, nil, err
	}
	return &highestVersion, obj, err
}

// Get works with a marshalled VersionedIDRef key. Direct usage should be avoided in favour of
// GetVersion or GetLatestVersion.
// Unlike the classic Get function it returns:
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

// GetVersion returns the stored object for the given VersionedIDRef.
// Unlike the classic Get function it returns:
//  - ErrNotFound when not found
//  - ErrDeleted when deleted
// Object won't be nil in success case

func (b VersioningBucket) GetVersion(db weave.ReadOnlyKVStore, ref VersionedIDRef) (Object, error) {
	key, err := ref.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal version id key")
	}
	return b.Get(db, key)
}

// Create stores the given data. It assigns an ID and initial version number to the object instance and returns the
// VersionedIDRef which won't be nil on success.
func (b VersioningBucket) Create(db weave.KVStore, data versionedData) (*VersionedIDRef, error) {
	newID, err := b.idGen.NextVal(db, data)
	if err != nil {
		return nil, err
	}
	return b.create(db, newID, data)
}

// CreateWithID stores the given data. It accepts an ID and assigns an initial version number to the object instance
// and returns the VersionedIDRef which won't be nil on success. This method is designed to be used for scenarios
// where an ID is needed to generate data within the entity before saving it.
func (b VersioningBucket) CreateWithID(db weave.KVStore, id []byte, data versionedData) (*VersionedIDRef, error) {
	if len(id) == 0 {
		return nil, errors.Wrap(errors.ErrEmpty, "id")
	}
	return b.create(db, id, data)
}

func (b VersioningBucket) create(db weave.KVStore, id []byte, data versionedData) (*VersionedIDRef, error) {
	if data.GetVersion() != 0 {
		return nil, errors.Wrap(errors.ErrInput, "version is set on create")
	}
	data.SetVersion(1)
	idRef := VersionedIDRef{ID: id, Version: data.GetVersion()}
	return b.safeUpdate(db, idRef, data)
}

// Deprecated: Save will always return an error. Use Create or Update instead.
func (b VersioningBucket) Save(db weave.KVStore, model Object) error {
	return errors.Wrap(errors.ErrHuman, "raw save not supported")
}

// Update persists the given data object with a new derived version key in the storage.
// The VersionedIDRef returned won't be nil on success and contains the new version number.
// The currentKey must be the latest one in usage or an ErrDuplicate is returned.
func (b VersioningBucket) Update(db weave.KVStore, id []byte, data versionedData) (*VersionedIDRef, error) {
	if data.GetVersion() == 0 {
		return nil, errors.Wrap(errors.ErrEmpty, "version not set")
	}
	currentKey := VersionedIDRef{ID: id, Version: data.GetVersion()}
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

// safeUpdate expects all validations have happened before
func (b VersioningBucket) safeUpdate(db weave.KVStore, newVersionKey VersionedIDRef, data CloneableData) (*VersionedIDRef, error) {
	key, err := newVersionKey.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshall versioned id ref")
	}
	// store new version
	return &newVersionKey, b.Bucket.Save(db, NewSimpleObj(key, data))
}

// Exists returns if an object is persisted for that given VersionedIDRef.
// If it points to the tombstone as deletion marker, ErrDeleted is returned.
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

// Delete stores a tombstone value for the new highest version. It will return this key on success.
// A version for the given ID must exists or ErrNotFound is returned.
// When already deleted Err
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

// marker is a null value type that satisfies CloneableData.
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
