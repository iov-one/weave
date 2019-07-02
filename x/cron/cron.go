package cron

import (
	"context"
	"encoding/binary"
	"reflect"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
)

// Schedule queues the transcation in the database to be executed at given
// time. Due to the implementation details, transaction is guaranteed to be
// executed after given time, but not exactly at given time.
// When successful, returns the scheduled task ID.
//
// If another transaction is already scheduled for the exact same time, execution
// of this transaction is delayed until the next free slot.
//
// Time granularity is second.
func Schedule(db weave.KVStore, runAt time.Time, tx weave.Tx) ([]byte, error) {
	const granularity = time.Second
	runAt = runAt.Round(granularity)

	rawTx, err := tx.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "marshal transaction")
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

		if err := db.Set(key, rawTx); err != nil {
			return nil, errors.Wrap(err, "cannot update queue")
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

// MsgCron allows to execute messages queued for future execution. It does this
// by implementing weave.Ticker interface.
type MsgCron struct {
	hn     weave.Handler
	txtype reflect.Type
}

// NewMsgCron returns a message cron instance that is using given handler to
// process all queued messages that execution time is due.
func NewMsgCron(tx weave.Tx, h weave.Handler) *MsgCron {
	return &MsgCron{
		hn:     h,
		txtype: reflect.TypeOf(tx),
	}
}

var _ weave.Ticker = (*MsgCron)(nil)

// Tick implementes weave.Ticker interface.
// Tick can process any number of messages suitable for execution. All changes
// are done atomically and apply only on success.
func (c *MsgCron) Tick(ctx context.Context, db store.CacheableKVStore) (*weave.TickResult, error) {
	now, err := weave.BlockTime(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get current time")
	}

	// TODO: collect results
	result := &weave.TickResult{}

	for {
		tx := reflect.New(c.txtype.Elem()).Interface().(weave.Tx)
		switch key, err := peek(db, now, tx); {
		case err == nil:
			// Each task is processed using its own cache instance
			// to ensure changes are atomic and task processing
			// independent.
			cache := db.CacheWrap()
			if _, err := c.hn.Deliver(ctx, cache, tx); err != nil {
				// Discard any changes that the deliver could
				// have created. We do not want to persist
				// those.
				cache.Discard()
			}

			// Remove the task from the queue as it was processed.
			// Do it via cache to keep it atomic.
			cache.Delete(key)
			if err := cache.Write(); err != nil {
				return result, errors.Wrap(err, "cannot write cache")
			}
		case errors.ErrEmpty.Is(err):
			// No more messages queued for execution at this time.
			return result, nil
		default:
			return result, errors.Wrap(err, "cannot pop queue")
		}
	}
}

// peek reads from the queue a single message that reached its execution time
// and returns it. It returns ErrEmpty if there is no message suitable for
// processing.
func peek(db weave.KVStore, now time.Time, dest weave.Tx) ([]byte, error) {
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
			return key, errors.Wrapf(err, "cannot unmarshal %q", key)
		}
		return key, nil
	case errors.ErrIteratorDone.Is(err):
		return nil, errors.ErrEmpty
	default:
		return nil, errors.Wrap(err, "cannot get next item")
	}
}
