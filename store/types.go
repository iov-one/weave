package store

import "github.com/confio/weave"

// Move references for all storage types into this package
// for shorter names everywhere

// ReadOnlyKVStore is an alias to interface in root package
type ReadOnlyKVStore = weave.ReadOnlyKVStore

// KVStore is an alias to interface in root package
type KVStore = weave.KVStore

// SetDeleter is an alias to interface in root package
type SetDeleter = weave.SetDeleter

// Iterator is an alias to interface in root package
type Iterator = weave.Iterator

// Batch is an alias to interface in root package
type Batch = weave.Batch

// CacheableKVStore is an alias to interface in root package
type CacheableKVStore = weave.CacheableKVStore

// KVCacheWrap is an alias to interface in root package
type KVCacheWrap = weave.KVCacheWrap

// CommitKVStore is an alias to interface in root package
type CommitKVStore = weave.CommitKVStore

// CommitID is an alias to interface in root package
type CommitID = weave.CommitID

// Model is an alias to interface in root package
type Model = weave.Model

// Pair is an alias to function in root package
var Pair = weave.Pair
