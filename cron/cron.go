package cron

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
)

// Put queues the message in the database to be executed at given time.
// Due to the implementation details, message is guaranteed to be executed
// after given time, but not exactly at given time.
//
// If another message is already scheduled for the exact same time, execution
// of this message is delayed until the next free slot.
//
// Time granularity is second.
func Put(db weave.KVStore, runAt time.Time, msg weave.Msg) error {
	const granularity = time.Second
	runAt = runAt.Round(granularity)

	msgb, err := msg.Marshal()
	if err != nil {
		return errors.Wrap(err, "marshal message")
	}

	for {
		key := queueKey(runAt)
		if ok, err := db.Has(key); err != nil {
			return errors.Wrap(err, "cannot check key existance")
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

		if err := db.Set(key, msgb); err != nil {
			return errors.Wrap(err, "cannot update queue")
		}
		return nil
	}
}

// pop removes from the queue a single message that reached its execution time
// and returns it. It returns ErrEmpty if there is no message suitable for
// processing.
func pop(db weave.KVStore, now time.Time) (weave.Msg, error) {
	// since := queueKey(time.Time{}) // Zero time is early enough.
	// until := queueKey(now)
	// it := db.Iterate(since, until)
	// ...

	// TODO: iterate from 0 a.d. till now and return the first message
	// encountered. Return ErrEmpty if no message was found.
	panic("not implemented")
}

func queueKey(t time.Time) []byte {
	rawTime := make([]byte, 8)
	binary.BigEndian.PutUint64(rawTime, uint64(t.UnixNano()))
	return append([]byte("_cronmsg:runat:"), rawTime...)
}

// MsgCron allows to execute messages queued for future execution. It does this
// by implementing weave.Ticker interface.
type MsgCron struct {
	hn weave.Handler
}

// NewMsgCron returns a message cron instance that is using given handler to
// process all queued messages that execution time is due.
func NewMsgCron(h weave.Handler) *MsgCron {
	return &MsgCron{
		hn: h,
	}
}

var _ weave.Ticker = (*MsgCron)(nil)

// Tick implementes weave.Ticker interface.
// Tick can process any number of messages suitable for execution. All changes
// are done atomically and apply only on success.
func (c *MsgCron) Tick(ctx context.Context, db store.CacheableKVStore) (*weave.TickResult, error) {
	result := &weave.TickResult{}

	for {
		// Each pop is using its own cache instance. This is to enforce
		// atomic pop and successful processing. Otherwise if only the
		// delivery was using cache it would be possible that even
		// though message was removed from the database and
		// successfully processed, we cannot write the result.
		taskCache := db.CacheWrap()
		msg, err := pop(taskCache, time.Now())
		switch {
		case err == nil:
			// Do not run Check as it is only to prevent spam.
			// Deliver must provide the same level of validation so
			// it is enough to call it alone.
			deliverCache := taskCache.CacheWrap()
			if _, err := c.hn.Deliver(ctx, deliverCache, &cronTx{msg: msg}); err != nil {
				deliverCache.Discard()
			} else if err := deliverCache.Write(); err != nil {
				// If deliver cache cannot be flushed, we do
				// not want to write the pop result to the
				// database as well as we want the task to be
				// processed again later.
				taskCache.Discard()
				continue
			}

			if err := taskCache.Write(); err != nil {
				return result, errors.Wrap(err, "cannot write task cache")
			}
		case errors.ErrEmpty.Is(err):
			// No more messages queued for execution at this time.
			return result, nil
		default:
			return result, errors.Wrap(err, "cannot pop queue")
		}
	}
}

// cronTx is a transaction implementation created for the cron ticker. It wraps
// a message so that it can be processed by the handler.
type cronTx struct {
	msg weave.Msg
}

var _ weave.Tx = (*cronTx)(nil)

func (tx *cronTx) GetMsg() (weave.Msg, error) {
	return tx.msg, nil
}

func (tx *cronTx) Marshal() ([]byte, error) {
	return nil, errors.Wrap(errors.ErrHuman, "operation not supported")
}

func (tx *cronTx) Unmarshal([]byte) error {
	return errors.Wrap(errors.ErrHuman, "operation not supported")
}
