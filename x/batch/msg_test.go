package batch_test

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/batch"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/mock"
)

func TestMsg(t *testing.T) {
	Convey("Test Validate", t, func() {
		msg := &mockMsg{}
		Convey("Test happy flow", func() {
			msg.On("MsgList").Return(make([]weave.Msg, 10), nil)
			So(batch.Validate(msg), ShouldBeNil)
			msg.AssertExpectations(t)
		})

		Convey("Test validation errors", func() {
			msg.On("MsgList").Return(make([]weave.Msg, 11), errors.ErrEmpty)
			err := errors.AsMulti(batch.Validate(msg))
			assert.Equal(t, errors.ErrEmpty.Is(err.Named("Message")), true)
			assert.Equal(t, errors.ErrInput.Is(err.Named("Size")), true)
			msg.AssertExpectations(t)
		})

	})
}

var _ batch.Msg = (*mockMsg)(nil)

type mockMsg struct {
	mock.Mock
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
	args := m.Mock.Called()
	return args.Error(0)
}

func (m *mockMsg) MsgList() ([]weave.Msg, error) {
	args := m.Mock.Called()
	return args.Get(0).([]weave.Msg), args.Error(1)
}
