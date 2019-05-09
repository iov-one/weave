package migration

import (
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
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

	db := store.MemStore()

	ensureSchemaVersion(t, db, thisPkgName, 1)

	handler := SchemaMigratingHandler(thisPkgName, &weavetest.Handler{})
	// Use custom register reference so that our test is not polluted by
	// extenrnal registrations.
	useHandlerRegister(t, handler, reg)

	var err error

	// Message has the same schema version as currently active one. No
	// migration should be applied.
	// Handler is modyfing/migrating message in place so we can use `msg1`
	// reference to check migrated message state.
	msg1 := &MyMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Content:  "foo",
	}
	_, err = handler.Check(nil, db, &weavetest.Tx{Msg: msg1})
	assert.Nil(t, err)
	assert.Equal(t, msg1.Metadata.Schema, uint32(1))
	assert.Equal(t, msg1.Content, "foo")
	_, err = handler.Deliver(nil, db, &weavetest.Tx{Msg: msg1})
	assert.Nil(t, err)
	assert.Equal(t, msg1.Metadata.Schema, uint32(1))
	assert.Equal(t, msg1.Content, "foo")

	// Upgrade the schema an ensure all further handler calls are migrating
	// the schema as well.
	ensureSchemaVersion(t, db, thisPkgName, 2)

	_, err = handler.Check(nil, db, &weavetest.Tx{Msg: msg1})
	assert.Nil(t, err)
	assert.Equal(t, msg1.Metadata.Schema, uint32(2))
	assert.Equal(t, msg1.Content, "foo m2")
	_, err = handler.Deliver(nil, db, &weavetest.Tx{Msg: msg1})
	assert.Nil(t, err)
	assert.Equal(t, msg1.Metadata.Schema, uint32(2))
	assert.Equal(t, msg1.Content, "foo m2")

	// If a message is already migrated, it must not be upgraded.
	msg2 := &MyMsg{
		Metadata: &weave.Metadata{Schema: 2},
		Content:  "bar",
	}
	_, err = handler.Check(nil, db, &weavetest.Tx{Msg: msg2})
	assert.Nil(t, err)
	assert.Equal(t, msg2.Metadata.Schema, uint32(2))
	assert.Equal(t, msg2.Content, "bar")
	_, err = handler.Deliver(nil, db, &weavetest.Tx{Msg: msg2})
	assert.Nil(t, err)
	assert.Equal(t, msg2.Metadata.Schema, uint32(2))
	assert.Equal(t, msg2.Content, "bar")
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

// useHandlerRegister set a custom migration register for a given
// schemaMigratingHandler. This function is needed to keep tests independent
// and avoid influencing one other by modifying the global migrations register.
func useHandlerRegister(t testing.TB, h weave.Handler, r *register) {
	t.Helper()
	handler, ok := h.(*schemaMigratingHandler)
	if !ok {
		t.Fatalf("only schemaMigratingHandler can use a register, got %T", h)
	}
	handler.migrations = r
}

func TestSchemaRoutingHandlerCannotBeEmpty(t *testing.T) {
	assert.Panics(t, func() {
		SchemaRoutingHandler(nil)
	})
}

func TestSchemaRoutingHandlerCannotRegisterZeroVersionHandler(t *testing.T) {
	assert.Panics(t, func() {
		SchemaRoutingHandler([]weave.Handler{
			0: &weavetest.Handler{},
		})
	})
}

func TestSchemaRoutingHandler(t *testing.T) {
	cases := map[string]struct {
		Tx      weave.Tx
		Handler weave.Handler
		WantErr *errors.Error

		// After handler call ensure that handler registered with given
		// schema version was called given amount of times. This is
		// possible because weavetest.Handler is counting method calls.
		//
		// Mapping of schema version->handler call count.
		WantCalls map[int]int
	}{
		"non migratable message": {
			Tx: &weavetest.Tx{
				Msg: &weavetest.Msg{},
			},
			Handler: SchemaRoutingHandler([]weave.Handler{
				1: &weavetest.Handler{},
			}),
			WantErr:   errors.ErrType,
			WantCalls: map[int]int{1: 0},
		},
		"route to handler by the exact match of message schema version": {
			Tx: &weavetest.Tx{
				Msg: &MigratableMsg{
					Metadata: &weave.Metadata{Schema: 2},
				},
			},
			Handler: SchemaRoutingHandler([]weave.Handler{
				1: &weavetest.Handler{},
				2: &weavetest.Handler{},
			}),
			WantCalls: map[int]int{
				1: 0,
				2: 1,
			},
		},
		"route to handler by selecting the highest available handler, but not higher than the schema version": {
			Tx: &weavetest.Tx{
				Msg: &MigratableMsg{
					Metadata: &weave.Metadata{Schema: 20},
				},
			},
			Handler: SchemaRoutingHandler([]weave.Handler{
				1:   &weavetest.Handler{},
				5:   &weavetest.Handler{},
				100: &weavetest.Handler{},
			}),
			WantCalls: map[int]int{
				1: 0,
				// 5 is the highest registered schema version
				// handler that is not higher than 20. It must
				// be used to route message with schema version
				// 20.
				5:   1,
				100: 0,
			},
		},
		"router with only high value schema handlers cannot route low version schema message": {
			Tx: &weavetest.Tx{
				Msg: &MigratableMsg{
					Metadata: &weave.Metadata{Schema: 4},
				},
			},
			Handler: SchemaRoutingHandler([]weave.Handler{
				// It is allowed to register handlers for
				// schema versions starting with a value
				// greater than one. In this case, routing
				// lower value schema message must fail.
				10: &weavetest.Handler{},
				14: &weavetest.Handler{},
			}),
			WantErr: errors.ErrSchema,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			_, err := tc.Handler.Deliver(nil, nil, tc.Tx)
			if !tc.WantErr.Is(err) {
				t.Fatalf("unexpected error result: %s", err)
			}
			for ver, wantCnt := range tc.WantCalls {
				schemaHandler := tc.Handler.(schemaRoutingHandler)
				cnt := schemaHandler[ver].(*weavetest.Handler).CallCount()
				if cnt != wantCnt {
					t.Errorf("for version %d handler want %d calls, got %d", ver, wantCnt, cnt)
				}
			}
		})
	}
}

type MigratableMsg struct {
	weavetest.Msg
	Metadata *weave.Metadata
}

var _ Migratable = (*MigratableMsg)(nil)

func (m *MigratableMsg) GetMetadata() *weave.Metadata {
	return m.Metadata
}
