package queue

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
func Put(db weave.KVStore, runAt time.Time, msg weave.Msg) error {
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
			runAt = runAt.Add(1)
			continue
		}

		if err := db.Set(key, msgb); err != nil {
			return errors.Wrap(err, "cannot update queue")
		}
		return nil
	}
}

// Pop removes from the queue a single message that reached its execution time
// and returns it. It returns ErrEmpty if there is no message suitable for
// processing.
func Pop(db weave.KVStore, now time.Time) (weave.Msg, error) {
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
	cache := db.CacheWrap()
	defer cache.Discard()

	for {
		msg, err := Pop(cache, time.Now())
		switch {
		case err == nil:
			// Do not run Check as it is only to prevent spam.
			// Deliver must provide the same level of validation so
			// it is enough to call it alone.
			if _, err := c.hn.Deliver(ctx, cache, &cronTx{msg: msg}); err != nil {
				return nil, errors.Wrapf(err, "cannot deliver %T message", msg)
			}
		case errors.ErrEmpty.Is(err):
			// No more messages queue for execution at this time.
			if err := cache.Write(); err != nil {
				return nil, errors.Wrap(err, "cache write")
			}
			return &weave.TickResult{}, nil
		default:
			return nil, errors.Wrap(err, "cannot pop queue")
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
