package currency

import (
	"github.com/iov-one/weave"
)

// Initializer fulfils the Initializer interface to load data from the genesis
// file
type Initializer struct{}

var _ weave.Initializer = (*Initializer)(nil)

// FromGenesis will parse initial account info from genesis and save it to the
// database
func (*Initializer) FromGenesis(opts weave.Options, params weave.GenesisParams, kv weave.KVStore) error {
	var tokens []struct {
		Ticker string `json:"ticker"`
		Name   string `json:"name"`
	}
	if err := opts.ReadOptions("currencies", &tokens); err != nil {
		return err
	}

	bucket := NewTokenInfoBucket()
	for _, t := range tokens {
		obj := NewTokenInfo(t.Ticker, t.Name)
		if err := bucket.Save(kv, obj); err != nil {
			return err
		}
	}
	return nil
}
