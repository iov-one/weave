package migration

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
)

// Initializer fulfils the InitStater interface to load data from
// the genesis file
type Initializer struct{}

var _ weave.Initializer = Initializer{}

// FromGenesis will parse initial account info from genesis
// and save it to the database
func (Initializer) FromGenesis(opts weave.Options, params weave.GenesisParams, kv weave.KVStore) error {
	if err := gconf.InitConfig(kv, opts, "migration", &Configuration{}); err != nil {
		return errors.Wrap(err, "migration config")
	}

	var packages []struct {
		Ver uint32 `json:"ver"`
		Pkg string `json:"pkg"`
	}
	if err := opts.ReadOptions("initialize_schema", &packages); err != nil {
		return errors.Wrap(err, "initialize schema")
	}

	b := NewSchemaBucket()

	// Before ensuring the schema of above packages is initialized force
	// register migration package schema.
	// This is solving a chicken-egg problem. We could not register any
	// schema version without Schema model being enabled (schema registered
	// with version one).
	_, err := b.Create(kv, &Schema{
		Metadata: &weave.Metadata{Schema: 1},
		Pkg:      "migration",
		Version:  1,
	})
	// Duplicated initializations are ignored.
	if err != nil && !errors.ErrDuplicate.Is(err) {
		return errors.Wrap(err, "initialize migration package schema")
	}

	for _, schema := range packages {
		for ver := uint32(1); ver <= schema.Ver; ver++ {
			_, err := b.Create(kv, &Schema{
				Metadata: &weave.Metadata{Schema: 1},
				Pkg:      schema.Pkg,
				Version:  ver,
			})
			// Duplicated initializations are ignored.
			if err != nil && !errors.ErrDuplicate.Is(err) {
				return errors.Wrapf(err, "initialize %q schema with version %d", schema.Pkg, ver)
			}
		}
	}

	return nil
}
