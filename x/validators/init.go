package validators

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

const (
	optKey = "update_validators"
)

// Initializer fulfils the InitStater interface to load data from
// the genesis file
type Initializer struct{}

var _ weave.Initializer = Initializer{}

// FromGenesis will parse initial account info from genesis
// and save it to the database
func (Initializer) FromGenesis(opts weave.Options, params weave.GenesisParams, kv weave.KVStore) error {
	var accounts WeaveAccounts
	if err := opts.ReadOptions(optKey, &accounts); err != nil {
		return errors.Wrap(err, "cannot read genesis options")
	}
	if err := accounts.Validate(); err != nil {
		return errors.Wrap(err, "accounts validation")
	}
	accts := AccountsWith(accounts)
	bucket := NewAccountBucket()
	if err := bucket.Save(kv, accts); err != nil {
		return errors.Wrap(err, "bucket save")
	}

	// Deduplicate validators for storage.
	vu := weave.ValidatorUpdatesFromABCI(params.Validators).Deduplicate(true)
	if err := vu.Validate(); err != nil {
		return errors.Wrap(err, "validator updates")
	}

	return errors.Wrap(weave.StoreValidatorUpdates(kv, vu), "store validator updates")
}
