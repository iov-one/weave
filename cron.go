package weave

import (
	"time"

	"github.com/tendermint/tendermint/libs/common"
)

// Ticker is an interface used to call background tasks scheduled for
// execution.
type Ticker interface {
	// Tick is a method called at the beginning of the block. It should be
	// used to execute any scheduled tasks.
	//
	// Because beginning of the block does not allow for an error response
	// this method does not return one as well. It is the implementation
	// responsibility to handle all error situations.
	// In case of an error that is an instance specific (ie database
	// issues) it might be neccessary for the method to terminate (ie
	// panic). An instance specific issue means that all other nodes most
	// likely succeeded processing the task and have different state than
	// this instance. This means that this node is out of sync with the
	// rest of the network and cannot continue operating as its state is
	// invalid.
	Tick(ctx Context, store CacheableKVStore) TickResult
}

// TickResult represents the result of a single tick run.
type TickResult struct {
	// Tags contains a list of tags that were produced during a single tick
	// execution. They should be included in the block that this tick
	// result was produced.
	// Empty tag list is a valid result.
	Tags []common.KVPair

	// Diff contains a list of validator update operations produced during
	// a single tick execution. They should be included in the block that
	// this tick result was produced.
	// Empty validator update list is a valid result.
	Diff []ValidatorUpdate
}

// Scheduler is an interface implemented to allow scheduling message execution.
type Scheduler interface {
	// Schedule queues given message in the database to be executed at
	// given time.  Message will be executed with context containing
	// provided authentication addresses.
	// When successful, returns the scheduled task ID.
	Schedule(KVStore, time.Time, []Condition, Msg) ([]byte, error)

	// Delete removes a scheduled task from the queue. It returns
	// ErrNotFound if task with given ID is not present in the queue.
	Delete(KVStore, []byte) error
}
