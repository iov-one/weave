package migration

import (
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestSchemaMigratingHandler(t *testing.T) {
	const thisPkgName = "testpkg"

	reg := newRegister()

	reg.MustRegister(1, &MyMsg{}, NoModification)
	reg.MustRegister(2, &MyMsg{}, func(db weave.ReadOnlyKVStore, m Migratable) error {
		msg := m.(*MyMsg)
		msg.Content += " m2"
		return msg.err
	})
	reg.MustRegister(3, &MyMsg{}, func(db weave.ReadOnlyKVStore, m Migratable) error {
		panic("not implemented")
	})

	db := store.MemStore()

	schema := NewSchemaBucket()
	if _, err := schema.Create(db, &Schema{Metadata: &weave.Metadata{Schema: 1}, Pkg: thisPkgName, Version: 1}); err != nil {
		t.Fatalf("cannot register schema version: %s", err)
	}

	handler := SchemaMigratingHandler(thisPkgName, &weavetest.Handler{})
	// Use custom register reference so that our test is not polluted by
	// extenrnal registrations.
	handler.(*schemaMigratingHandler).migrations = reg

	var err error

	// Message has the same schema version as currently active one. No
	// migration should be applied.
	// Handler is modyfing/migrating message in place so we can use `msg`
	// reference to check migrated message state.
	msg := &MyMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Content:  "foo",
	}
	_, err = handler.Check(nil, db, &weavetest.Tx{Msg: msg})
	assert.Nil(t, err)
	assert.Equal(t, msg.Metadata.Schema, uint32(1))
	assert.Equal(t, msg.Content, "foo")
	_, err = handler.Deliver(nil, db, &weavetest.Tx{Msg: msg})
	assert.Nil(t, err)
	assert.Equal(t, msg.Metadata.Schema, uint32(1))
	assert.Equal(t, msg.Content, "foo")

	// Upgrade the schema an ensure all further handler calls are migrating
	// the schema as well.
	if _, err := schema.Create(db, &Schema{Metadata: &weave.Metadata{Schema: 1}, Pkg: thisPkgName, Version: 2}); err != nil {
		t.Fatalf("cannot register schema version: %s", err)
	}

	_, err = handler.Check(nil, db, &weavetest.Tx{Msg: msg})
	assert.Nil(t, err)
	assert.Equal(t, msg.Metadata.Schema, uint32(2))
	assert.Equal(t, msg.Content, "foo m2")
	_, err = handler.Deliver(nil, db, &weavetest.Tx{Msg: msg})
	assert.Nil(t, err)
	assert.Equal(t, msg.Metadata.Schema, uint32(2))
	assert.Equal(t, msg.Content, "foo m2")

	// If a message is already migrated, it must not be upgraded.
	msg = &MyMsg{
		Metadata: &weave.Metadata{Schema: 2},
		Content:  "bar",
	}
	_, err = handler.Check(nil, db, &weavetest.Tx{Msg: msg})
	assert.Nil(t, err)
	assert.Equal(t, msg.Metadata.Schema, uint32(2))
	assert.Equal(t, msg.Content, "bar")
	_, err = handler.Deliver(nil, db, &weavetest.Tx{Msg: msg})
	assert.Nil(t, err)
	assert.Equal(t, msg.Metadata.Schema, uint32(2))
	assert.Equal(t, msg.Content, "bar")
}

type MyMsg struct {
	Metadata *weave.Metadata
	Content  string

	err error
}

func (msg *MyMsg) GetMetadata() *weave.Metadata {
	return msg.Metadata
}

func (msg *MyMsg) Validate() error {
	if err := msg.Metadata.Validate(); err != nil {
		return err
	}
	return msg.err
}

func (msg *MyMsg) Marshal() ([]byte, error) {
	return json.Marshal(msg)
}

func (msg *MyMsg) Unmarshal(raw []byte) error {
	return json.Unmarshal(raw, &msg)
}

func (MyMsg) Path() string {
	return "testpkg/mymsg"
}

var _ Migratable = (*MyMsg)(nil)
var _ weave.Msg = (*MyMsg)(nil)
