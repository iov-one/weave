package cron

import (
	"context"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

func TestQueue(t *testing.T) {
	now := time.Now()
	db := store.MemStore()

	if _, err := Schedule(db, now.Add(-5*time.Second), &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "test/1"}}, nil); err != nil {
		t.Fatalf("cannot schedule first message: %s", err)
	}
	if _, err := Schedule(db, now.Add(-5*time.Second), &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "test/2"}}, nil); err != nil {
		t.Fatalf("cannot schedule second message: %s", err)
	}
	if _, err := Schedule(db, now.Add(-10*time.Second), &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "test/3"}}, nil); err != nil {
		t.Fatalf("cannot schedule third message: %s", err)
	}

	var tx weavetest.Tx
	if _, err := peek(db, now.Add(-time.Hour), &tx); !errors.ErrEmpty.Is(err) {
		t.Logf("%#v", tx.Msg)
		t.Fatalf("want no task, got %+v", err)
	}

	// Order of scheduing (from the "oldest") should be [3, 1, 2].
	// 1 and 2 have the same execution time but 1 was scheduled first.
	wantPaths := []string{
		"test/3",
		"test/1",
		"test/2",
	}
	for _, want := range wantPaths {
		var tx weavetest.Tx
		key, err := peek(db, now, &tx)
		if err != nil {
			t.Fatalf("want task with message path %q, got %+v", want, err)
		}
		db.Delete(key)
		if got := tx.Msg.Path(); got != want {
			t.Fatalf("want %q message path, got %q", want, got)
		}
	}
}

func TestTicker(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	now := time.Now()

	db := store.MemStore()

	migration.MustInitPkg(db, "cron")

	msg1 := &weavetest.Msg{RoutePath: "test/1"}
	if _, err := Schedule(db, now, &weavetest.Tx{Msg: msg1}, nil); err != nil {
		t.Fatalf("cannot schedule message: %s", err)
	}

	msg2 := &weavetest.Msg{RoutePath: "test/2"}
	// Second message scheduled at the same time must be processed as the
	// second.
	if _, err := Schedule(db, now, &weavetest.Tx{Msg: msg2}, nil); err != nil {
		t.Fatalf("cannot schedule message: %s", err)
	}

	handler := &cronHandler{}
	cron := NewMsgCron(&weavetest.Tx{}, handler)

	ctx = weave.WithBlockTime(ctx, now.Add(time.Hour))
	if _, err := cron.tick(ctx, db); err != nil {
		t.Fatalf("cannot tick: %s", err)
	}

	if want, got := 2, len(handler.delivered); want != got {
		t.Fatalf("want %d tasks processed, got %d", want, got)
	}
	if want, got := msg1.Path(), handler.delivered[0].Path(); want != got {
		t.Fatalf("want %q message path to be delivered, got %q", want, got)
	}
	if want, got := msg2.Path(), handler.delivered[1].Path(); want != got {
		t.Fatalf("want %q message path to be delivered, got %q", want, got)
	}
}

type cronHandler struct {
	delivered []weave.Msg
	res       weave.DeliverResult
	err       error
}

func (cronHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	panic("cron must not call check")
}

func (h *cronHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := tx.GetMsg()
	if err != nil {
		panic("cannot get message")
	}
	h.delivered = append(h.delivered, msg)
	// copy
	res := h.res
	return &res, h.err
}
