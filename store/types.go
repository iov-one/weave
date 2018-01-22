//nolint
package store

import "github.com/confio/weave"

// Move references for all storage types into this package
// for shorter names everywhere

type KVStore = weave.KVStore
type Iterator = weave.Iterator
type Batch = weave.Batch
type CacheableKVStore = weave.CacheableKVStore
type KVCacheWrap = weave.KVCacheWrap
type CommitKVStore = weave.CommitKVStore
type CommitID = weave.CommitID
