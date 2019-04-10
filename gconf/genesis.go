package gconf

import (
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

	//forname, value := range conf {
	//	panic("todo")
	//}

	return nil
}
