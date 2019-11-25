package datamigration

import (
	"context"
	"testing"

	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
)

func TestMigrationCanBeRegisteredOnce(t *testing.T) {
	reg := newRegister()

	noop := func(context.Context, weave.KVStore) error { return nil }
	sigs := []weave.Address{weavetest.NewCondition().Address()}

	if err := reg.Register("migration name", Migration{ChainID: "chain-a", RequiredSigners: sigs, Migrate: noop}); err != nil {
		t.Fatalf("unexpected failure: %+v", err)
	}

	// Double registration
	if err := reg.Register("migration name", Migration{ChainID: "chain-a", RequiredSigners: sigs, Migrate: noop}); !errors.ErrState.Is(err) {
		t.Fatalf("expected ErrState, got %+v", err)
	}
}
