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
		raw, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("cannot serialize %s: %s", name, err)
		}
		key := []byte("gconf:" + name)
		db.Set(key, raw)
	}

	return nil
}
