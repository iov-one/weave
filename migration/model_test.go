package migration

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
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

func TestCurrentSchema(t *testing.T) {
	db := store.MemStore()

	packages := map[string]uint32{
		"one":    1,
		"four":   4,
		"seven":  7,
		"eleven": 11,
	}

	for name, ver := range packages {
		ensureSchemaVersion(t, db, name, ver)
	}

	b := NewSchemaBucket()

	for name, ver := range packages {
		if v, err := b.CurrentSchema(db, name); err != nil {
			t.Errorf("cannot get %q package schema version: %d", name, err)
		} else if v != ver {
			t.Errorf("invalid %q package schema version: %d", name, v)
		}
	}

	if _, err := b.CurrentSchema(db, "does-not-exist"); !errors.ErrNotFound.Is(err) {
		t.Fatalf("unexpected schema error of an unknown package: %s", err)
	}
}

// ensureSchemaVersion will ensure that all schema versions up to given one are
// present. This activates schema version with given value and additionally all
// previous ones.
// This function fails the test is schema version cannot be registered.
// Duplicated registrations are ignored and not fatal.
func ensureSchemaVersion(t testing.TB, db weave.KVStore, pkgName string, version uint32) {
	t.Helper()

	b := NewSchemaBucket()
	for v := uint32(1); v <= version; v++ {
		_, err := b.Create(db, &Schema{
			Metadata: &weave.Metadata{Schema: 1},
			Pkg:      pkgName,
			Version:  v,
		})
		if err != nil && !errors.ErrDuplicate.Is(err) {
			t.Fatalf("cannot register %d schema: %s", v, err)
		}
	}
}
