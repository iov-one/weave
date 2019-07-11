package cron

import (
	"bytes"
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
	"github.com/tendermint/tendermint/libs/common"
)

func TestSchedulerDelete(t *testing.T) {
	db := store.MemStore()
	now := time.Now()

	enc := NewTestTaskMarshaler(&weavetest.Msg{})
	s := NewScheduler(enc)

	if err := s.Delete(db, []byte("task-with-this-id-does-not-exist")); !errors.ErrNotFound.Is(err) {
		t.Fatalf("deletion of a non existing task failed with an unexpected error: %s", err)
	}

	tid, err := s.Schedule(db, now, nil, &weavetest.Msg{})
	if err != nil {
		t.Fatalf("cannot schedule a task: %s", err)
	}
	if err := s.Delete(db, tid); err != nil {
		t.Fatalf("cannot delete a scheduled task: %s", err)
	}
	if err := s.Delete(db, tid); !errors.ErrNotFound.Is(err) {
		t.Fatalf("double deletion failed with an unexpected error: %s", err)
	}

	// Use nil as handler so that it panics if used. This ticker is not
	// able to successfully process a task.
	ticker := NewTicker(nil, enc)

	ctx := context.Background()
	ctx = weave.WithBlockTime(ctx, now.Add(time.Hour))
	ctx = weave.WithHeight(ctx, 123)
	// Tick must process no tasks. It will crash this process if any task
	// is to be processed because we use nil handler.
	ticker.Tick(ctx, db)
}

func TestRoundT(t *testing.T) {
	cases := map[string]struct {
		input       time.Time
		granularity time.Duration
		want        time.Time
	}{
		"the same time when zero granularity": {
			input:       time.Unix(12345679, 0),
			granularity: 0,
			want:        time.Unix(12345679, 0),
		},
		"the same time when the input requires no rounding": {
			input:       time.Unix(100, 0),
			granularity: time.Second,
			want:        time.Unix(100, 0),
		},
		"round up when the cut off is more than half": {
			input:       time.Unix(101, 99),
			granularity: 2 * time.Second,
			want:        time.Unix(102, 0),
		},
		"round up when the cut off is less than half": {
			input:       time.Unix(100, 1),
			granularity: 2 * time.Second,
			want:        time.Unix(102, 0),
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			got := roundT(tc.input, tc.granularity)
			if !got.Equal(tc.want) {
				t.Fatalf("got %d, %d (%q)", got.Unix(), got.UnixNano(), got)
			}
		})
	}
}

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
	now := time.Now()

	var (
		conditionA = weavetest.NewCondition()
		conditionB = weavetest.NewCondition()
	)

	type task struct {
		id []byte // assigned during the runtime by the scheduler

		RunAt           time.Time
		Auth            []weave.Condition
		Msg             weavetest.Msg
		WantExec        bool
		WantExecSuccess bool
	}

	cases := map[string]struct {
		Tasks         []*task
		WantTickerErr *errors.Error
		Handler       cronHandler
	}{
		"no tasks": {
			Tasks:         nil,
			WantTickerErr: nil,
		},
		"authentication is provided and successful": {
			Tasks: []*task{
				{
					RunAt: now.Add(-time.Hour),
					Auth: []weave.Condition{
						conditionA,
						conditionB,
					},
					Msg:             weavetest.Msg{RoutePath: "test/1"},
					WantExec:        true,
					WantExecSuccess: true,
				},
			},
			Handler: cronHandler{
				wantAuth: []weave.Condition{conditionA, conditionB},
			},
		},
		"when authentication is missing message cannot be processed": {
			Tasks: []*task{
				{
					RunAt: now.Add(-time.Hour),
					Auth: []weave.Condition{
						conditionA,
						// Condition B is missing.
					},
					Msg:             weavetest.Msg{RoutePath: "test/1"},
					WantExec:        true,
					WantExecSuccess: false,
				},
			},
			Handler: cronHandler{
				wantAuth: []weave.Condition{conditionA, conditionB},
			},
		},
		"tasks that are not due yet": {
			Tasks: []*task{
				{
					RunAt:    now.Add(time.Hour),
					Auth:     nil,
					Msg:      weavetest.Msg{RoutePath: "test/1"},
					WantExec: false,
				},
				{
					RunAt:    now.Add(time.Hour),
					Auth:     nil,
					Msg:      weavetest.Msg{RoutePath: "test/2"},
					WantExec: false,
				},
			},
			WantTickerErr: nil,
		},
		"all tasks are due and successful": {
			Tasks: []*task{
				{
					RunAt:           now.Add(-time.Hour),
					Auth:            nil,
					Msg:             weavetest.Msg{RoutePath: "test/1"},
					WantExec:        true,
					WantExecSuccess: true,
				},
				{
					RunAt:           now.Add(-time.Hour),
					Auth:            nil,
					Msg:             weavetest.Msg{RoutePath: "test/2"},
					WantExec:        true,
					WantExecSuccess: true,
				},
			},
			WantTickerErr: nil,
		},
		"a task is due and failed": {
			Tasks: []*task{
				{
					RunAt:           now.Add(-time.Hour),
					Auth:            nil,
					Msg:             weavetest.Msg{RoutePath: "test/1"},
					WantExec:        true,
					WantExecSuccess: false,
				},
			},
			WantTickerErr: nil,
			Handler: cronHandler{
				errs: map[string]error{
					"test/1": errors.ErrState,
				},
			},
		},
		"a mixture of tasks, some are due and some are successful": {
			Tasks: []*task{
				{
					RunAt:    now.Add(time.Hour),
					Auth:     nil,
					Msg:      weavetest.Msg{RoutePath: "wait/1"},
					WantExec: false,
				},
				{
					RunAt:           now.Add(-time.Hour),
					Auth:            nil,
					Msg:             weavetest.Msg{RoutePath: "due/success"},
					WantExec:        true,
					WantExecSuccess: true,
				},
				{
					RunAt:           now.Add(-time.Hour),
					Auth:            nil,
					Msg:             weavetest.Msg{RoutePath: "due/failure"},
					WantExec:        true,
					WantExecSuccess: false,
				},
				{
					RunAt:    now.Add(time.Hour),
					Auth:     nil,
					Msg:      weavetest.Msg{RoutePath: "wait/2"},
					WantExec: false,
				},
			},
			WantTickerErr: nil,
			Handler: cronHandler{
				errs: map[string]error{
					"due/success": nil,
					"due/failure": errors.ErrHuman,
				},
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, "cron")
			enc := NewTestTaskMarshaler(&weavetest.Msg{})
			scheduler := NewScheduler(enc)
			ticker := NewTicker(&tc.Handler, enc)

			for i, task := range tc.Tasks {
				tid, err := scheduler.Schedule(db, task.RunAt, task.Auth, &task.Msg)
				if err != nil {
					t.Fatalf("cannot schedule #%d (%q) task: %s", i, task.Msg.Path(), err)
				}
				task.id = tid
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			ctx = weave.WithBlockTime(ctx, now)
			ctx = weave.WithHeight(ctx, 999)

			// Use tick instead of Tick method so that the error is
			// returned instead of terminating the process.
			// We need to tick only once in order to process all
			// tasks that are due.
			tags, _, err := ticker.tick(ctx, db)
			if !tc.WantTickerErr.Is(err) {
				t.Fatalf("unexpected ticker error: %+v", err)
			}

			for i, task := range tc.Tasks {
				wasExec := containsPairValue(tags, task.id)
				if wasExec != task.WantExec {
					t.Logf("%d tags created: %q", len(tags), tags)
					t.Fatalf("task #%d (%q) unexpected execution state: %v", i, task.Msg.Path(), wasExec)
				}
			}

			// For every executed task, there must be a state information persisted.
			b := NewTaskResultBucket()
			for i, task := range tc.Tasks {
				if !task.WantExec {
					continue
				}
				var tr TaskResult
				if err := b.One(db, task.id, &tr); err != nil {
					t.Fatalf("expected #%d (%q) task result, got %s", i, task.Msg.Path(), err)
				}
				if tr.Successful != task.WantExecSuccess {
					t.Fatalf("exected task #%d (%q) to be success=%v: %q", i, task.Msg.Path(), task.WantExecSuccess, tr.Info)
				}
			}
		})
	}
}

func containsPairValue(pairs []common.KVPair, item []byte) bool {
	for _, p := range pairs {
		if bytes.Equal(p.Value, item) {
			return true
		}
	}
	return false
}

type cronHandler struct {
	res weave.DeliverResult
	// Map message path to error that delivery of this message should
	// return.
	errs map[string]error

	auth Authenticator
	// When provided, conditions are required to authenticate.
	wantAuth []weave.Condition
}

func (cronHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	panic("cron must not call check")
}

func (h *cronHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := tx.GetMsg()
	if err != nil {
		panic("cannot get message")
	}

	for _, c := range h.wantAuth {
		if !h.auth.HasAddress(ctx, c.Address()) {
			err := errors.Wrapf(errors.ErrUnauthorized,
				"%q condition (%q) not found", c, c.Address())
			return nil, err
		}
	}

	// copy
	res := h.res
	return &res, h.errs[msg.Path()]
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
