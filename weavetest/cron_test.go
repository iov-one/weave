package weavetest

import (
	"testing"
	"time"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
)

func TestCron(t *testing.T) {
	var c Cron

	db := store.MemStore()

	tid, err := c.Schedule(db, time.Now(), nil, nil)
	if err != nil {
		t.Fatalf("cannot schedule a task: %s", err)
	}
	if err := c.Delete(db, tid); err != nil {
		t.Fatalf("cannot delete a task: %s", err)
	}

	if err := c.Delete(db, tid); !errors.ErrNotFound.Is(err) {
		t.Fatalf("want ErrNotFound, got %+v", err)
	}
}
