package gconf

import (
	"encoding/json"
	"fmt"

	"github.com/iov-one/weave"
)

// Initializer fulfils the InitStater interface to load data from
// the genesis file
type Initializer struct{}

var _ weave.Initializer = Initializer{}

// FromGenesis will parse initial account info from genesis
// and save it to the database
func (Initializer) FromGenesis(opts weave.Options, db weave.KVStore) error {
	conf := make(map[string]interface{})
	err := opts.ReadOptions("gconf", &conf)
	if err != nil {
		return err
	}

	for name, value := range conf {
		if err := SetValue(db, name, value); err != nil {
			return err
		}
	}

	return nil
}

// SetValue sets given value for the configuration property. Value must be JSON
// serializable.
// Usually this function is not needed, because genesis allows to load all at
// once. But it comes in handy when writing tests and only few configuration
// values are necessary.
func SetValue(db weave.KVStore, propName string, value interface{}) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cannot serialize %s: %s", propName, err)
	}
	key := []byte("gconf:" + propName)
	db.Set(key, raw)
	return nil
}
