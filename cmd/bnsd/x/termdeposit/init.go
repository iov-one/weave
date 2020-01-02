package termdeposit

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
)

// Initializer fulfils the Initializer interface to load data from the genesis
// file
type Initializer struct{}

var _ weave.Initializer = (*Initializer)(nil)

// FromGenesis will parse initial account info from genesis and save it to the
// database
func (*Initializer) FromGenesis(opts weave.Options, params weave.GenesisParams, db weave.KVStore) error {
	conf := Configuration{
		Metadata: &weave.Metadata{Schema: 1},
	}
	switch err := gconf.InitConfig(db, opts, "termdeposit", &conf); {
	default:
		// All good.
	case errors.ErrNotFound.Is(err):
		return nil
	case err != nil:
		return errors.Wrap(err, "cannot initialize gconf based configuration")
	}

	var contracts []struct {
		ValidSince weave.UnixTime `json:"valid_since"`
		ValidUntil weave.UnixTime `json:"valid_until"`
		Rate       Frac           `json:"rate"`
	}

	if err := opts.ReadOptions("depositcontract", &contracts); err != nil {
		return err
	}
	b := NewDepositContractBucket()
	for i, c := range contracts {
		contract := DepositContract{
			Metadata:   &weave.Metadata{Schema: 1},
			ValidSince: c.ValidSince,
			ValidUntil: c.ValidUntil,
		}
		if err := contract.Validate(); err != nil {
			return errors.Wrapf(err, "contract %d is invalid", i)
		}
		if _, err := b.Put(db, nil, &contract); err != nil {
			return errors.Wrapf(err, "store contract %d", i)
		}
	}
	return nil
}
