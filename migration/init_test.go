package migration

import (
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
)

func TestGenesisInitializeSchemaVersions(t *testing.T) {
	const genesis = `
	{
		"conf": {
			"migration": {
				"admin": "6a4832947079b0a851ec4daa3dae69de1f7741eb"
			}
		},
		"initialize_schema": ["c", "b", "a"]
	}
	`

	var opts weave.Options
	if err := json.Unmarshal([]byte(genesis), &opts); err != nil {
		t.Fatalf("cannot unmarshal genesis: %s", err)
	}

	db := store.MemStore()
	var ini Initializer
	if err := ini.FromGenesis(opts, db); err != nil {
		t.Fatalf("cannot load genesis: %s", err)
	}

	wantSchemaVersions := []string{
		"a", "b", "c",

		// Running the initializer must always ensure the migration
		// package schema version is at least one.
		"migration",
	}
	for _, pkgName := range wantSchemaVersions {
		ver, err := NewSchemaBucket().CurrentSchema(db, pkgName)
		if err != nil {
			t.Fatalf("cannot get current schema for %q package: %s", pkgName, err)
		}
		if ver != 1 {
			t.Fatalf("unexpected schema version for %q package: %d", pkgName, ver)
		}
	}
}
