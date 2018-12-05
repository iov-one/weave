package token

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
)

// Initializer fulfils the Initializer interface to load data from the genesis
// file
type Initializer struct{}

var _ weave.Initializer = (*Initializer)(nil)

// FromGenesis will parse initial account info from genesis and save it to the
// database
func (*Initializer) FromGenesis(opts weave.Options, db weave.KVStore) error {
	var tokens []struct {
		Ticker  string
		Name    string
		SigFigs int32
	}
	if err := opts.ReadOptions("tokens", &tokens); err != nil {
		return err
	}

	bucket := NewTokenInfoBucket()
	for _, t := range tokens {
		obj := orm.NewSimpleObj([]byte(t.Ticker), &TokenInfo{
			Name:    t.Name,
			SigFigs: t.SigFigs,
		})
		if err := bucket.Save(db, obj); err != nil {
			return err
		}
	}
	return nil
}
