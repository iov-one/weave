package migration

import (
	"testing"

	"github.com/iov-one/weave/store"
)

func TestMustInitPkgDuplication(t *testing.T) {
	db := store.MemStore()

	MustInitPkg(db, "mypkg")

	ver, err := NewSchemaBucket().CurrentSchema(db, "mypkg")
	if err != nil {
		t.Fatalf("cannot get mypkg schema version: %s", err)
	}
	if ver != 1 {
		t.Fatalf("want schema initialized with version 1, got %d", ver)
	}

	// It can be called any number of times. If already initialized, this
	// must be a noop call.
	MustInitPkg(db, "mypkg")
	MustInitPkg(db, "mypkg")
}
