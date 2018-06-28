package update_validators

import (
	"github.com/confio/weave"
)

const optKey = "update_validators"

// Initializer fulfils the InitStater interface to load data from
// the genesis file
type Initializer struct{}

var _ weave.Initializer = Initializer{}

// FromGenesis will parse initial account info from genesis
// and save it to the database
func (Initializer) FromGenesis(opts weave.Options, kv weave.KVStore) error {
	accounts := WeaveAccounts{}
	err := opts.ReadOptions(optKey, &accounts)
	if err != nil {
		return err
	}
	bucket := NewBucket()
	for _, addr := range accounts.Addresses {
		if err := addr.Validate(); err != nil {
			return err
		}
		wallet := AccountsWith(accounts)

		err := bucket.Save(kv, wallet)
		if err != nil {
			return err
		}
	}
	return nil
}
