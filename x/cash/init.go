package cash

import (
	"github.com/iov-one/weave"
)

const optKey = "cash"

// GenesisAccount is used to parse the json from genesis file
// use weave.Address, so address in hex, not base64
type GenesisAccount struct {
	Address weave.Address `json:"address"`
	Set
}

// Initializer fulfils the InitStater interface to load data from
// the genesis file
type Initializer struct{}

var _ weave.Initializer = Initializer{}

// FromGenesis will parse initial account info from genesis
// and save it to the database
func (Initializer) FromGenesis(opts weave.Options, kv weave.KVStore) error {
	accts := []GenesisAccount{}
	err := opts.ReadOptions(optKey, &accts)
	if err != nil {
		return err
	}
	bucket := NewBucket()
	for _, acct := range accts {
		if err := acct.Address.Validate(); err != nil {
			return err
		}
		wallet, err := WalletWith(acct.Address, acct.Set.Coins...)
		if err != nil {
			return err
		}
		err = bucket.Save(kv, wallet)
		if err != nil {
			return err
		}
	}
	return nil
}
