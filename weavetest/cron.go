package weavetest

import (
	"bytes"
	"crypto/rand"
	"sort"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

// Cron is a in memory implementation of the ticker and scheduler.
type Cron struct {
	Err   error
	tasks []*crontask
}

type crontask struct {
	tid   []byte
	runAt time.Time
	auth  []weave.Condition
	msg   weave.Msg
}

var _ weave.Scheduler = (*Cron)(nil)
var _ weave.Ticker = (*Cron)(nil)

// Schedule implementes weave.Scheduler interface.
func (c *Cron) Schedule(db weave.KVStore, runAt time.Time, auth []weave.Condition, msg weave.Msg) ([]byte, error) {
	if c.Err != nil {
		return nil, c.Err
	}

	tid := make([]byte, 8)
	if _, err := rand.Read(tid); err != nil {
		panic(err)
	}

	c.tasks = append(c.tasks, &crontask{
		runAt: runAt,
		auth:  auth,
		msg:   msg,
	})

	// Keep in order from the oldest to the newest. Those to be executed
	// first are first.
	sort.Slice(c.tasks, func(i, j int) bool {
		return c.tasks[i].runAt.Before(c.tasks[j].runAt)
	})

	return tid, nil
}

// Delete implementes weave.Scheduler interface.
func (c *Cron) Delete(db weave.KVStore, taskID []byte) error {
	if c.Err != nil {
		return c.Err
	}

	for i, t := range c.tasks {
		if bytes.Equal(t.tid, taskID) {
			c.tasks = append(c.tasks[:i], c.tasks[i+1:]...)
			return nil
		}
	}
	return errors.Wrap(errors.ErrNotFound, "no task")
}

// Tick implementes weave.Ticker interface.
func (c *Cron) Tick(ctx weave.Context, store weave.CacheableKVStore) [][]byte {
	now, err := weave.BlockTime(ctx)
	if err != nil {
		panic(err)
	}

	var executed [][]byte
	for _, t := range c.tasks {
		if !t.runAt.After(now) {
			executed = append(executed, t.tid)
		} else {
			// Tasks are ordered by execution time.
			break
		}
	}
	c.tasks = c.tasks[len(executed):]
	return executed
}
