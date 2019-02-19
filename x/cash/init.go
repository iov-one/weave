package cash

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
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

// FeeInitializer fulfils the FeeInitializer interface to load data from the genesis
// file
type FeeInitializer struct{}

var _ weave.Initializer = (*FeeInitializer)(nil)

// FromGenesis will parse initial account info from genesis and save it to the
// database
func (*FeeInitializer) FromGenesis(opts weave.Options, db weave.KVStore) error {
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
