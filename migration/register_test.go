package migration

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestApply(t *testing.T) {
	reg := newRegister()
	reg.MustRegister(0, &MyMsg{}, NoModyfication)
	reg.MustRegister(1, &MyMsg{}, func(ctx weave.Context, db weave.KVStore, p Payload) error {
		msg := p.(*MyMsg)
		msg.Content += "to1"
		return nil
	})
	reg.MustRegister(2, &MyMsg{}, NoModyfication)
	reg.MustRegister(3, &MyMsg{}, func(ctx weave.Context, db weave.KVStore, p Payload) error {
		msg := p.(*MyMsg)
		msg.Content += "to3"
		return nil
	})

	mymsg := &MyMsg{
		Header:  &weave.Header{},
		Content: "zero ",
	}

	// Running a migration can bring it up to any state in the future.
	assert.Nil(t, reg.Apply(nil, nil, mymsg, 2))
	assert.Equal(t, mymsg.Header.Schema, uint32(2))
	assert.Equal(t, mymsg.Content, "zero to1")

	assert.Nil(t, reg.Apply(nil, nil, mymsg, 3))
	assert.Equal(t, mymsg.Header.Schema, uint32(3))
	assert.Equal(t, mymsg.Content, "zero to1to3")
}

func TestMigrateUnknownVersion(t *testing.T) {
	reg := newRegister()
	reg.MustRegister(0, &MyMsg{}, NoModyfication)
	reg.MustRegister(1, &MyMsg{}, NoModyfication)
	reg.MustRegister(2, &MyMsg{}, NoModyfication)

	mymsg := &MyMsg{
		Header:  &weave.Header{},
		Content: "zero ",
	}

	// Migration attempt to a non existing version must fail. It will
	// upgrade the message to the highest available state.
	if err := reg.Apply(nil, nil, mymsg, 999); !errors.ErrInvalidState.Is(err) {
		t.Fatalf("unexpected migration failure: %s", err)
	}
	assert.Equal(t, mymsg.Header.Schema, uint32(2))
}

type MyMsg struct {
	Header *weave.Header
	VErr   error

	Content string
}

func (msg *MyMsg) GetHeader() *weave.Header {
	return msg.Header
}

func (msg *MyMsg) Validate() error {
	return msg.VErr
}

var _ Payload = (*MyMsg)(nil)
