package cron

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

func TestTaskQueue(t *testing.T) {
	now := time.Now()
	db := store.MemStore()

	enc := NewTestTaskMarshaler(&weavetest.Msg{})
	s := NewScheduler(enc)

	if _, err := s.Schedule(db, now.Add(-5*time.Second), nil, &weavetest.Msg{RoutePath: "test/1"}); err != nil {
		t.Fatalf("cannot schedule first message: %s", err)
	}
	if _, err := s.Schedule(db, now.Add(-5*time.Second), nil, &weavetest.Msg{RoutePath: "test/2"}); err != nil {
		t.Fatalf("cannot schedule second message: %s", err)
	}
	if _, err := s.Schedule(db, now.Add(-10*time.Second), nil, &weavetest.Msg{RoutePath: "test/3"}); err != nil {
		t.Fatalf("cannot schedule third message: %s", err)
	}

	if key, _, err := peek(db, now.Add(-time.Hour)); !errors.ErrEmpty.Is(err) {
		t.Logf("key: %q", key)
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
		key, raw, err := peek(db, now)
		if err != nil {
			t.Fatalf("want task with message path %q, got %+v", want, err)
		}
		_, msg, err := enc.UnmarshalTask(raw)
		if err != nil {
			t.Fatalf("cannot unmarshal task: %s", err)
		}
		db.Delete(key)
		if got := msg.Path(); got != want {
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

	enc := NewTestTaskMarshaler(&weavetest.Msg{})
	handler := &cronHandler{}
	scheduler := NewScheduler(enc)
	ticker := NewTicker(handler, enc)

	msg1 := &weavetest.Msg{RoutePath: "test/1"}
	if _, err := scheduler.Schedule(db, now, nil, msg1); err != nil {
		t.Fatalf("cannot schedule message: %s", err)
	}

	msg2 := &weavetest.Msg{RoutePath: "test/2"}
	// Second message scheduled at the same time must be processed as the
	// second.
	if _, err := scheduler.Schedule(db, now, nil, msg2); err != nil {
		t.Fatalf("cannot schedule message: %s", err)
	}

	ctx = weave.WithBlockTime(ctx, now.Add(time.Hour))
	if _, err := ticker.tick(ctx, db); err != nil {
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

// NewTestTaskMarshaler returns a TaskMarshaler implementation that supports
// only a single message type.
func NewTestTaskMarshaler(emptyMsg weave.Msg) *testTaskMarshaler {
	return &testTaskMarshaler{
		msgType: reflect.TypeOf(emptyMsg),
	}
}

type testTaskMarshaler struct {
	msgType reflect.Type
}

var _ TaskMarshaler = (*testTaskMarshaler)(nil)

func (t *testTaskMarshaler) MarshalTask(auth []weave.Condition, msg weave.Msg) ([]byte, error) {
	if reflect.TypeOf(msg) != t.msgType {
		return nil, errors.Wrap(errors.ErrType, "unsupported message type")
	}
	rawMsg, err := msg.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "cannot marshal message")
	}
	return json.Marshal(serializedTask{
		Auth:   auth,
		RawMsg: rawMsg,
	})

}

func (t *testTaskMarshaler) UnmarshalTask(raw []byte) ([]weave.Condition, weave.Msg, error) {
	var st serializedTask
	if err := json.Unmarshal(raw, &st); err != nil {
		return nil, nil, errors.Wrap(err, "cannot JSON deserialize task")
	}
	msg := reflect.New(t.msgType.Elem()).Interface().(weave.Msg)
	if err := msg.Unmarshal(st.RawMsg); err != nil {
		return nil, nil, errors.Wrap(err, "cannot deserialize msg")
	}
	return st.Auth, msg, nil
}

type serializedTask struct {
	Auth   []weave.Condition
	RawMsg []byte
}
