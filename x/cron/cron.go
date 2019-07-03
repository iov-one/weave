package cron

import (
	"context"
	"encoding/binary"
	"fmt"
	"reflect"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
)

// Task is an interface that must be implemented by user of this package. Only
// a single implementation must be used at any time.
type Task interface {
	// Task implements weave transaction interface as it is to be processed
	// by handler. It is up to the user of this package to define what a
	// task is. It is not recommended to use the same structure for
	// synchronous transactions and asynchronous tasks.
	weave.Tx

	// In addition to the weave.Tx interface, task MAY implement
	// ContextPreparer to ensure context variables before execution.
}

// ContextPreparer is an interface that can be implemented by the Task
// implementation. If that is the case, Prepare method is used to enrich the
// context before passing it to the handler to execute the task.
//
// This is a way for the task to take care of persisting and than restoring any
// information that should be present in the context during the execution.
type ContextPreparer interface {
	// Prepare the context to be used for this task execution.
	Prepare(ctx context.Context) context.Context
}

// Schedule queues the transcation in the database to be executed at given
// time. Transaction will be executed with context containing provided
// authentication addresses.
// When successful, returns the scheduled task ID.
//
// Due to the implementation details, transaction is guaranteed to be executed
// after given time, but not exactly at given time.  If another transaction is
// already scheduled for the exact same time, execution of this transaction is
// delayed until the next free slot.
//
// Time granularity is second.
func Schedule(db weave.KVStore, runAt time.Time, task Task) ([]byte, error) {
	const granularity = time.Second
	runAt = runAt.Round(granularity)

	rawTask, err := task.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "marshal task")
	}

	for {
		key := queueKey(runAt)
		if ok, err := db.Has(key); err != nil {
			return nil, errors.Wrap(err, "cannot check key existance")
		} else if ok {
			// If the key is already in use, instead of storing a
			// list of messages under each key, which is a very
			// unlikely to happen, increase the execution time by
			// the smallest duration.
			// Message is guaranteed to be executed not earlier
			// than given time, NOT at exactly given time.
			runAt = runAt.Add(granularity)
			continue
		}

		if err := db.Set(key, rawTask); err != nil {
			return nil, errors.Wrap(err, "cannot store in queue")
		}
		return key, nil
	}
}

func queueKey(t time.Time) []byte {
	rawTime := make([]byte, 8)
	// Zero time does not need to put any data as the bytes are already set
	// to zero.
	if !t.IsZero() {
		binary.BigEndian.PutUint64(rawTime, uint64(t.UnixNano()))
	}
	return append([]byte("_crontx:runat:"), rawTime...)
}

// Ticker allows to execute messages queued for future execution. It does this
// by implementing weave.Ticker interface.
type Ticker struct {
	hn       weave.Handler
	taskType reflect.Type
	results  orm.ModelBucket
}

// NewTicker returns a cron instance that is using given handler to process all
// queued messages that execution time is due.
//
// Task is an instance of a Task interface implementation. This is the
// implementation that the returned ticker can work with and no other
// implementation is allowed. All data serialization and processing will be
// done assuming provided type.
func NewTicker(emptyInstance Task, h weave.Handler) *Ticker {
	return &Ticker{
		hn:       h,
		taskType: reflect.TypeOf(emptyInstance),
		results:  NewTaskResultBucket(),
	}
}

var _ weave.Ticker = (*Ticker)(nil)

// Tick implementes weave.Ticker interface.
// Tick can process any number of messages suitable for execution. All changes
// are done atomically and apply only on success.
func (t *Ticker) Tick(ctx context.Context, db store.CacheableKVStore) [][]byte {
	executed, err := t.tick(ctx, db)
	if err != nil {
		// This is a hopeless state. This error is most likely due to a
		// database issues or some other instance specific problems.
		// This problem is unique to this instance and this operation
		// most likely succeeded on other nodes. This means that there
		// is no way we could continue operating as this instance is
		// out of sync with the rest of the network.
		failTask(err)
	}
	return executed
}

// failTask is a variable so that it can be overwritten for tests.
var failTask = func(err error) {
	panic(fmt.Sprintf("%+v", err))
}

// tick process any number of tasks. It always returns a response and might
// return an error. This method is similar to the Tick except it provides an
// error. This makes it easier for the tests to check the result.
func (t *Ticker) tick(ctx context.Context, db store.CacheableKVStore) ([][]byte, error) {
	var executed [][]byte

	now, err := weave.BlockTime(ctx)
	if err != nil {
		return executed, errors.Wrap(err, "cannot get current time")
	}

	for {
		task := reflect.New(t.taskType.Elem()).Interface().(Task)
		switch key, err := peek(db, now, task); {
		case err == nil:
			res := TaskResult{
				Metadata:   &weave.Metadata{Schema: 1},
				Successful: true,
			}

			taskCtx := ctx
			if prep, ok := task.(ContextPreparer); ok {
				taskCtx = prep.Prepare(taskCtx)
			}

			// Each task is processed using its own cache instance
			// to ensure changes are atomic and task processing
			// independent.
			cache := db.CacheWrap()
			if _, err := t.hn.Deliver(taskCtx, cache, task); err != nil {
				// Discard any changes that the deliver could
				// have created. We do not want to persist
				// those.
				cache.Discard()
				res.Successful = false
				res.Info = err.Error()
			}

			if _, err := t.results.Put(cache, key, &res); err != nil {
				// Keep it atomic.
				cache.Discard()
				return executed, errors.Wrap(err, "cannot store result")
			}

			// Remove the task from the queue as it was processed.
			// Do it via cache to keep it atomic.
			cache.Delete(key)
			if err := cache.Write(); err != nil {
				return executed, errors.Wrap(err, "cannot write cache")
			}

			// Only when the database state is updated we can
			// consider this task executed. Otherwise any change is
			// being discarded.
			executed = append(executed, key)
		case errors.ErrEmpty.Is(err):
			// No more messages queued for execution at this time.
			return executed, nil
		default:
			return executed, errors.Wrap(err, "cannot pop queue")
		}
	}
}

// peek loads from the queue a single task that reached its execution time and
// returns its ID. It returns ErrEmpty if there is no message suitable for
// processing.
// Task are consumed in order of execution time, starting with the oldest.
func peek(db weave.KVStore, now time.Time, dest Task) ([]byte, error) {
	since := queueKey(time.Time{}) // Zero time is early enough.
	until := queueKey(now)
	it, err := db.Iterator(since, until)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create iterator")
	}
	defer it.Release()

	switch key, value, err := it.Next(); {
	case err == nil:
		if err := dest.Unmarshal(value); err != nil {
			return key, errors.Wrapf(err, "cannot unmarshal transaction %q", key)
		}
		return key, nil
	case errors.ErrIteratorDone.Is(err):
		return nil, errors.ErrEmpty
	default:
		return nil, errors.Wrap(err, "cannot get next item")
	}
}
