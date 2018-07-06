package validators

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

	err = accounts.Validate()
	if err != nil {
		return err
	}

	accts := AccountsWith(accounts)

	err = bucket.Save(kv, accts)
	if err != nil {
		return err
	}

	return nil
}
