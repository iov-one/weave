package cron

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x/utils"
	"github.com/tendermint/tendermint/libs/common"
)

// TaskMarshaler represents an encoded that is used to marshal and unmarshal a
// task. This interface is to be implemented by this package user.
type TaskMarshaler interface {
	// MarshalTask serialize given data into its binary format.
	MarshalTask(auth []weave.Condition, msg weave.Msg) ([]byte, error)

	// UnmarshalTask deserialize data (created using MarshalTask method)
	// from its binary representation into Go structures.
	UnmarshalTask([]byte) (auth []weave.Condition, msg weave.Msg, err error)
}

// NewScheduler returns a scheduler implementation that is using given encoding
// for serializing data. Returned scheduler implements weave.Scheduler
// interface.
//
// Always use the same marshaler for ticker and scheduler.
func NewScheduler(enc TaskMarshaler) *Scheduler {
	return &Scheduler{enc: enc}
}

// Scheduler is the weave.Scheduler implementation.
type Scheduler struct {
	enc TaskMarshaler
}

var _ weave.Scheduler = (*Scheduler)(nil)

// Schedule implements weave.Scheduler interface.
//
// Due to the implementation details, transaction is guaranteed to be executed
// after given time, but not exactly at given time. If another transaction is
// already scheduled for the exact same time, execution of this transaction is
// delayed until the next free slot.
//
// Time granularity is second.
func (s *Scheduler) Schedule(db weave.KVStore, runAt time.Time, auth []weave.Condition, msg weave.Msg) ([]byte, error) {
	const granularity = time.Second
	runAt = roundT(runAt, granularity)

	raw, err := s.enc.MarshalTask(auth, msg)
	if err != nil {
		return nil, errors.Wrap(err, "marshal task")
	}

	// We use execution time as the queue value and to keep it unique, we
	// increment the execution time if a taks is already scheduled at given
	// moment.
	// As a security measure add a limit for how many keys can be tried to
	// avoid a potential spam issue.
	const maxTries = 100
	for try := 0; try < maxTries; try++ {
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

		if err := db.Set(key, raw); err != nil {
			return nil, errors.Wrap(err, "cannot store in queue")
		}
		return key, nil
	}
	return nil, errors.Wrap(errors.ErrState, "too many tasks scheduled for this time")
}

// roundT returns given time, rounded up to given granularity. Returned time is
// never before the given one.
func roundT(t time.Time, granularity time.Duration) time.Time {
	if granularity == 0 {
		return t
	}
	rounded := t.Round(granularity)
	if rounded.Before(t) {
		rounded = rounded.Add(granularity)
	}
	return rounded
}

func queueKey(t time.Time) []byte {
	rawTime := make([]byte, 8)
	// Zero time does not need to put any data as the bytes are already set
	// to zero.
	if !t.IsZero() {
		binary.BigEndian.PutUint64(rawTime, uint64(t.UnixNano()))
	}
	return append([]byte("_crontask:runat:"), rawTime...)
}

// Delete implements weave.Scheduler interface.
func (s *Scheduler) Delete(db weave.KVStore, taskID []byte) error {
	if ok, err := db.Has(taskID); err != nil {
		return errors.Wrap(err, "failed to check existence of key")
	} else if !ok {
		return errors.Wrap(errors.ErrNotFound, "no task")
	}
	if err := db.Delete(taskID); err != nil {
		return errors.Wrap(err, "cannot delete")
	}
	return nil
}

// NewTicker returns a cron runner instance that is using given handler to
// process all queued messages that execution time is due. All serialization is
// done using provided marshaler.
//
// Always use the same marshaler for ticker and scheduler.
func NewTicker(h weave.Handler, enc TaskMarshaler) *Ticker {
	return &Ticker{
		hn:      h,
		enc:     enc,
		results: NewTaskResultBucket(),
	}
}

// Ticker allows to execute messages queued for future execution. It does this
// by implementing weave.Ticker interface.
type Ticker struct {
	hn      weave.Handler
	enc     TaskMarshaler
	results orm.ModelBucket
}

var _ weave.Ticker = (*Ticker)(nil)

// Tick implementes weave.Ticker interface.
//
// Tick can process any number of messages suitable for execution. All changes
// are done atomically and apply only on success.
func (t *Ticker) Tick(ctx context.Context, db store.CacheableKVStore) weave.TickResult {
	tags, vDiff, err := t.tick(ctx, db)
	if err != nil {
		// This is a hopeless state. This error is most likely due to a
		// database issues or some other instance specific problems.
		// This problem is unique to this instance and this operation
		// most likely succeeded on other nodes. This means that there
		// is no way we could continue operating as this instance is
		// out of sync with the rest of the network.
		failTask(err)
	}
	return weave.TickResult{
		Tags: tags,
		Diff: vDiff,
	}
}

// failTask is a variable so that it can be overwritten for tests.
var failTask = func(err error) {
	panic(fmt.Sprintf(`

Asynchronous task failed.

This error is most likely due to a database issues or some other instance
specific problems. This problem is unique to this instance and this operation
most likely succeeded on other nodes. This means that there is no way we could
continue operating as this instance is out of sync with the rest of the
network.

%+v

	`, err))
}

// tick process any number of tasks. It always returns a response and might
// return an error. This method is similar to the Tick except it provides an
// error. This makes it easier for the tests to check the result.
//
// Because this method can process multiple tasks at once, even when this
// method returns an error instance, returned values may contain a valid
// infromation other than the error as well. Always process returned tags and
// validator diffs regardless the returned error value.
func (t *Ticker) tick(ctx context.Context, db store.CacheableKVStore) ([]common.KVPair, []weave.ValidatorUpdate, error) {
	var (
		tags  []common.KVPair
		vDiff []weave.ValidatorUpdate
	)

	now, err := weave.BlockTime(ctx)
	if err != nil {
		return tags, vDiff, errors.Wrap(err, "cannot get current time")
	}
	blockHeight, ok := weave.GetHeight(ctx)
	if !ok {
		return tags, vDiff, errors.Wrap(errors.ErrState, "cannot get current block height")
	}

	// A safety measure to not execute too many tasks in one go. Maintain
	// an upper limit as we can process the rest of the queue in another
	// run.
	const maxExecuted = 50
	for proc := 0; proc < maxExecuted; proc++ {
		switch key, raw, err := peek(db, now); {
		case err == nil:
			// Each task is processed using its own cache instance
			// to ensure changes are atomic and task processing
			// independent.
			cache := db.CacheWrap()

			var (
				taskTags []common.KVPair
				taskDiff []weave.ValidatorUpdate
			)
			res := TaskResult{
				Metadata:   &weave.Metadata{Schema: 1},
				Successful: true,
				ExecTime:   weave.AsUnixTime(now),
				ExecHeight: blockHeight,
			}

			auth, msg, err := t.enc.UnmarshalTask(raw)
			if err != nil {
				res.Successful = false
				res.Info = fmt.Sprintf("cannot unmarshal task: %+v", err)
			} else {
				taskCtx := withAuth(ctx, auth)
				tx := &taskTx{msg: msg}
				// Execute deliver using the savepoint wrapper,
				// so that changes are applied to the cache
				// only when the Deliver call is successful.
				sp := utils.NewSavepoint().OnDeliver()
				if r, err := sp.Deliver(taskCtx, cache, tx, t.hn); err != nil {
					res.Successful = false
					res.Info = err.Error()
				} else {
					taskTags = r.Tags
					taskDiff = r.Diff
				}
			}

			if _, err := t.results.Put(cache, key, &res); err != nil {
				cache.Discard()
				return tags, vDiff, errors.Wrap(err, "cannot store result")
			}

			// Remove the task from the queue as it was processed.
			if err := cache.Delete(key); err != nil {
				cache.Discard()
				return tags, vDiff, errors.Wrap(err, "cannot delete task")
			}
			if err := cache.Write(); err != nil {
				cache.Discard()
				return tags, vDiff, errors.Wrap(err, "cannot write cache")
			}

			// Only when the database state is updated we can
			// consider this task executed. Otherwise any change is
			// being discarded and the task is not part of the
			// result.
			tags = append(tags, taskTags...)
			tags = append(tags, common.KVPair{
				Key:   []byte("cron"),
				Value: key,
			})
			vDiff = append(vDiff, taskDiff...)
		case errors.ErrEmpty.Is(err):
			// No more messages queued for execution at this time.
			return tags, vDiff, nil
		default:
			return tags, vDiff, errors.Wrap(err, "cannot pop queue")
		}
	}

	return tags, vDiff, nil
}

// peek reads from the queue a single task that reached its execution time and
// returns it encoded value and ID. It returns ErrEmpty if there is no message
// suitable for processing.
// Tasks are consumed in order of execution time, starting with the oldest.
func peek(db weave.KVStore, now time.Time) (id, raw []byte, err error) {
	since := queueKey(time.Time{}) // Zero time is early enough.
	until := queueKey(now)
	it, err := db.Iterator(since, until)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot create iterator")
	}
	defer it.Release()

	switch key, value, err := it.Next(); {
	case err == nil:
		return key, value, nil
	case errors.ErrIteratorDone.Is(err):
		return nil, nil, errors.ErrEmpty
	default:
		return nil, nil, errors.Wrap(err, "cannot get next item")
	}
}

// taskTx is a weave.Tx implementation created for running
// asynchronous tasks. It is a thin wrapper over the message.
type taskTx struct {
	msg weave.Msg
}

var _ weave.Tx = (*taskTx)(nil)

// GetMsg implements weave.Tx interface.
func (tx *taskTx) GetMsg() (weave.Msg, error) {
	return tx.msg, nil
}

// Unmarshal implements weave.Tx interface.
func (tx *taskTx) Unmarshal([]byte) error {
	return errors.Wrap(errors.ErrHuman, "operation not supported, task transaction is not serializable")
}

// Marshal implements weave.Tx interface.
func (tx *taskTx) Marshal() ([]byte, error) {
	return nil, errors.Wrap(errors.ErrHuman, "operation not supported, task transaction is not serializable")
}
