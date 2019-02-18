package fee

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
)

// Initializer fulfils the Initializer interface to load data from the genesis
// file
type Initializer struct{}

var _ weave.Initializer = (*Initializer)(nil)

// FromGenesis will parse initial account info from genesis and save it to the
// database
func (*Initializer) FromGenesis(opts weave.Options, db weave.KVStore) error {
	var fees []struct {
		Id  string `json:"id"`
		Fee x.Coin `json:"fee"`
	}

	if err := opts.ReadOptions("fees", &fees); err != nil {
		return err
	}

	// always default to IOV token
	for k, v := range fees {
		v.Fee.Ticker = "IOV"
		fees[k] = v
	}

	bucket := NewTransactionFeeBucket()
	for _, f := range fees {
		obj := NewTransactionFee(f.Id, f.Fee)
		if err := bucket.Save(db, obj); err != nil {
			return err
		}
	}

	return nil
}
