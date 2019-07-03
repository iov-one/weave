package batch_test

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/batch"
)

func TestMsgValidate(t *testing.T) {
	specs := map[string]struct {
		msg batch.Msg
		err *errors.Error
	}{
		"Happy flow": {
			msg: &mockMsg{list: make([]weave.Msg, 10)},
		},
		"Test list too long": {
			msg: &mockMsg{list: make([]weave.Msg, batch.MaxBatchMessages+1)},
			err: errors.ErrInput,
		},
		"Test error": {
			msg: &mockMsg{list: make([]weave.Msg, batch.MaxBatchMessages), listErr: errors.ErrState},
			err: errors.ErrState,
		},
	}

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			err := batch.Validate(spec.msg)
			if spec.err == nil {
				assert.Equal(t, nil, err)
				return
			}
			if !spec.err.Is(err) {
				t.Fatalf("epected error does not match: %v  but got %+v", spec.err, err)
			}
		})
	}

}

var _ batch.Msg = (*mockMsg)(nil)

type mockMsg struct {
	valErr  error
	listErr error
	list    []weave.Msg
}

func (m *mockMsg) Marshal() ([]byte, error) {
	panic("implement me")
}

func (m *mockMsg) Unmarshal([]byte) error {
	panic("implement me")
}

func (m *mockMsg) Path() string {
	panic("implement me")
}

func (m *mockMsg) Validate() error {
	return m.valErr
}

func (m *mockMsg) MsgList() ([]weave.Msg, error) {
	return m.list, m.listErr
}
