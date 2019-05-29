package store

// Recorder interface is implemented by anything returned from
// NewRecordingStore
type Recorder interface {
	KVPairs() map[string][]byte
}

// NewRecordingStore initializes a recording store wrapping this
// base store, using cached alternative if possible
//
// We need to expose this optional functionality through the interface
// wrapper so downstream components (like Savepoint) can use reflection
// to CacheWrap.
func NewRecordingStore(db KVStore) KVStore {
	changes := make(map[string][]byte)
	if cached, ok := db.(CacheableKVStore); ok {
		return &cacheableRecordingStore{
			CacheableKVStore: cached,
			changes:          changes,
		}
	}
	return &recordingStore{
		KVStore: db,
		changes: changes,
	}
}

//------- non-cached recording store

// recordingStore wraps a normal KVStore and records any change operations
type recordingStore struct {
	KVStore
	// changes is a map from key to (recordSet|recordDelete)
	changes map[string][]byte
}

var _ KVStore = (*recordingStore)(nil)

// KVPairs returns the content of changes as KVPairs
// Key is the merkle store key that changes.
// Value is "s" or "d" for set or delete.
// TODO: don't use map as return value!!
func (r *recordingStore) KVPairs() map[string][]byte {
	return r.changes
}

// Set records the changes while performing
func (r *recordingStore) Set(key, value []byte) error {
	r.changes[string(key)] = value
	return r.KVStore.Set(key, value)
}

// Delete records the changes while performing
func (r *recordingStore) Delete(key []byte) error {
	r.changes[string(key)] = nil
	return r.KVStore.Delete(key)
}

// NewBatch makes sure all writes go through this one
func (r *recordingStore) NewBatch() Batch {
	return &recorderBatch{
		changes: r.changes,
		b:       r.KVStore.NewBatch(),
	}
}

//------- cached recording store

// cacheableRecordingStore wraps a CacheableKVStore
// and records any change operations
type cacheableRecordingStore struct {
	CacheableKVStore
	// changes is a map from key to (recordSet|recordDelete)
	changes map[string][]byte
}

var _ CacheableKVStore = (*cacheableRecordingStore)(nil)

// KVPairs returns the content of changes as KVPairs
// Key is the merkle store key that changes.
// Value is the value written (for set), or nil (for delete)
// TODO: don't use map as return value!!
func (r *cacheableRecordingStore) KVPairs() map[string][]byte {
	return r.changes
}

// Set records the changes while performing
//
// TODO: record new value???
func (r *cacheableRecordingStore) Set(key, value []byte) error {
	r.changes[string(key)] = value
	return r.CacheableKVStore.Set(key, value)
}

// Delete records the changes while performing
func (r *cacheableRecordingStore) Delete(key []byte) error {
	r.changes[string(key)] = nil
	return r.CacheableKVStore.Delete(key)
}

// NewBatch makes sure all writes go through this one
func (r *cacheableRecordingStore) NewBatch() Batch {
	return &recorderBatch{
		changes: r.changes,
		b:       r.CacheableKVStore.NewBatch(),
	}
}

// CacheWrap makes sure all cached writes also go through this
func (r *cacheableRecordingStore) CacheWrap() KVCacheWrap {
	// TODO: reuse FreeList between multiple cache wraps....
	// We create/destroy a lot per tx when processing a block
	return NewBTreeCacheWrap(r, r.NewBatch(), nil)
}

//----- batch recording, write to changes map from Recorder

type recorderBatch struct {
	changes map[string][]byte
	b       Batch
}

var _ Batch = (*recorderBatch)(nil)

func (r *recorderBatch) Set(key, value []byte) error {
	r.changes[string(key)] = value
	return r.b.Set(key, value)
}

// Delete records the changes while performing
func (r *recorderBatch) Delete(key []byte) error {
	r.changes[string(key)] = nil
	return r.b.Delete(key)
}

func (r *recorderBatch) Write() error {
	return r.b.Write()
}
