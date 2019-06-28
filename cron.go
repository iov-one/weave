package weave

import "time"

// Ticker is an interface used to call background tasks scheduled for
// execution.
type Ticker interface {
	// Tick is a method called at the beginning of the block. It should be
	// used to execute any scheduled tasks.
	//
	// Returned is always the list of task IDs that were executed. A task
	// is considered executed when processing it caused any change to the
	// state (even if it is only removing the task from the queue and no
	// other change).
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
	Tick(ctx Context, store CacheableKVStore) (taskIDs [][]byte)
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
