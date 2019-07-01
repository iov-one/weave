package cron

import (
	"context"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

func TestCron(t *testing.T) {
	now := time.Now()

	db := store.MemStore()

	msg1 := &weavetest.Msg{RoutePath: "test/1"}
	if err := Schedule(db, now, &weavetest.Tx{Msg: msg1}); err != nil {
		t.Fatalf("cannot schedule message: %s", err)
	}

	msg2 := &weavetest.Msg{RoutePath: "test/2"}
	// Second message scheduled at the same time must be processed as the
	// second.
	if err := Schedule(db, now, &weavetest.Tx{Msg: msg2}); err != nil {
		t.Fatalf("cannot schedule message: %s", err)
	}

	handler := &cronHandler{}
	cron := NewMsgCron(&weavetest.Tx{}, handler)
	cron.now = func() time.Time { return now.Add(time.Hour) }

	_, err := cron.Tick(context.Background(), db)
	if err != nil {
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
