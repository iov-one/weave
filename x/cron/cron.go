package cron

import (
	"context"
	"encoding/binary"
	"reflect"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
)

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
func Schedule(
	db weave.KVStore,
	runAt time.Time,
	tx weave.Tx,
	auth []weave.Address,
) ([]byte, error) {
	const granularity = time.Second
	runAt = runAt.Round(granularity)

	rawTx, err := tx.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "marshal transaction")
	}

	task := &Task{
		Metadata:   &weave.Metadata{Schema: 1},
		Serialized: rawTx,
		Auth:       auth,
	}
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
	hn      weave.Handler
	txtype  reflect.Type
	results orm.ModelBucket
}

// NewMsgCron returns a message cron instance that is using given handler to
// process all queued messages that execution time is due.
func NewMsgCron(tx weave.Tx, h weave.Handler) *Ticker {
	return &Ticker{
		hn:      h,
		txtype:  reflect.TypeOf(tx),
		results: NewTaskResultBucket(),
	}
}

var _ weave.Ticker = (*Ticker)(nil)

// Tick implementes weave.Ticker interface.
// Tick can process any number of messages suitable for execution. All changes
// are done atomically and apply only on success.
func (t *Ticker) Tick(ctx context.Context, db store.CacheableKVStore) [][]byte {
	executed, err := t.tick(ctx, db)
	if err != nil {
		// TODO log error
	}
	return executed
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
		tx := reflect.New(t.txtype.Elem()).Interface().(weave.Tx)
		switch key, auth, err := peek(db, now, tx); {
		case err == nil:
			info := TaskResult{
				Metadata:   &weave.Metadata{Schema: 1},
				Successful: true,
			}
			// Each task is processed using its own cache instance
			// to ensure changes are atomic and task processing
			// independent.
			cache := db.CacheWrap()
			if _, err := t.hn.Deliver(ctx, cache, tx); err != nil {
				// Discard any changes that the deliver could
				// have created. We do not want to persist
				// those.
				cache.Discard()
				info.Successful = false
				info.Info = err.Error()
			}

			if _, err := t.results.Put(cache, key, &info); err != nil {
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

// peek reads from the queue a single message that reached its execution time
// and returns it. It returns ErrEmpty if there is no message suitable for
// processing.
func peek(db weave.KVStore, now time.Time, dest weave.Tx) ([]byte, []weave.Address, error) {
	since := queueKey(time.Time{}) // Zero time is early enough.
	until := queueKey(now)
	it, err := db.Iterator(since, until)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot create iterator")
	}
	defer it.Release()

	switch key, value, err := it.Next(); {
	case err == nil:
		var task Task

		if err := task.Unmarshal(value); err != nil {
			return key, nil, errors.Wrapf(err, "cannot unmarshal task %q", key)
		}

		if err := dest.Unmarshal(task.Serialized); err != nil {
			return key, task.Auth, errors.Wrapf(err, "cannot unmarshal transaction %q", key)
		}
		return key, task.Auth, nil
	case errors.ErrIteratorDone.Is(err):
		return nil, nil, errors.ErrEmpty
	default:
		return nil, nil, errors.Wrap(err, "cannot get next item")
	}
}
