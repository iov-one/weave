package migration

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestZeroMigrationIsNotAllowed(t *testing.T) {
	reg := newRegister()

	if err := reg.Register(0, &MyMsg{}, NoModification); !errors.ErrInput.Is(err) {
		t.Fatalf("unexpected invalid version registration error: %s", err)
	}
	if err := reg.Apply(nil, &MyMsg{}, 0); !errors.ErrInput.Is(err) {
		t.Fatalf("unexpected invalid version registration error: %s", err)
	}
}

func TestRegisterMigrationMustBeSequential(t *testing.T) {
	reg := newRegister()

	// Each migration must start with 1.
	if err := reg.Register(2, &MyMsg{}, NoModification); !errors.ErrInput.Is(err) {
		t.Fatalf("unexpected error when missing previous migration: %s", err)
	}

	reg.MustRegister(1, &MyMsg{}, NoModification)
	reg.MustRegister(2, &MyMsg{}, NoModification)

	if err := reg.Register(4, &MyMsg{}, NoModification); !errors.ErrInput.Is(err) {
		t.Fatalf("unexpected error when missing previous migration: %s", err)
	}

	reg.MustRegister(3, &MyMsg{}, NoModification)
	reg.MustRegister(4, &MyMsg{}, NoModification)
}

func TestApply(t *testing.T) {
	reg := newRegister()
	reg.MustRegister(1, &MyMsg{}, NoModification)
	reg.MustRegister(2, &MyMsg{}, func(db weave.ReadOnlyKVStore, m Migratable) error {
		msg := m.(*MyMsg)
		msg.Content += "to2"
		return nil
	})
	reg.MustRegister(3, &MyMsg{}, NoModification)
	reg.MustRegister(4, &MyMsg{}, func(db weave.ReadOnlyKVStore, m Migratable) error {
		msg := m.(*MyMsg)
		msg.Content += "to4"
		return nil
	})

	mymsg := &MyMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Content:  "init ",
	}

	// Running a migration can bring it up to any state in the future.
	assert.Nil(t, reg.Apply(nil, mymsg, 3))
	assert.Equal(t, mymsg.Metadata.Schema, uint32(3))
	assert.Equal(t, mymsg.Content, "init to2")

	assert.Nil(t, reg.Apply(nil, mymsg, 4))
	assert.Equal(t, mymsg.Metadata.Schema, uint32(4))
	assert.Equal(t, mymsg.Content, "init to2to4")
}

func TestMigrateUnknownVersion(t *testing.T) {
	reg := newRegister()
	reg.MustRegister(1, &MyMsg{}, NoModification)
	reg.MustRegister(2, &MyMsg{}, NoModification)
	reg.MustRegister(3, &MyMsg{}, NoModification)

	mymsg := &MyMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Content:  "init ",
	}

	// Migration attempt to a non existing version must fail. It will
	// upgrade the message to the highest available state.
	if err := reg.Apply(nil, mymsg, 999); !errors.ErrSchema.Is(err) {
		t.Fatalf("unexpected migration failure: %s", err)
	}
	assert.Equal(t, mymsg.Metadata.Schema, uint32(3))
}
