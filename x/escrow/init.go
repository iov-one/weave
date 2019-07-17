package escrow

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/x/cash"
	"github.com/pkg/errors"
)

var _ weave.Initializer = (*Initializer)(nil)

// Initializer fulfils the Initializer interface to load data from the genesis file
type Initializer struct {
	Minter cash.CoinMinter
}

// FromGenesis will parse initial escrow  info from genesis and save it in the database.
func (i *Initializer) FromGenesis(opts weave.Options, params weave.GenesisParams, kv weave.KVStore) error {
	var escrows []struct {
		Source      weave.Address  `json:"source"`
		Arbiter     weave.Address  `json:"arbiter"`
		Destination weave.Address  `json:"destination"`
		Timeout     weave.UnixTime `json:"timeout"`
		Amount      []*coin.Coin   `json:"amount"`
	}

	if err := opts.ReadOptions("escrow", &escrows); err != nil {
		return err
	}
	bucket := NewBucket()
	for _, e := range escrows {
		key, err := escrowSeq.NextVal(kv)
		if err != nil {
			return errors.Wrap(err, "cannot acquire key")
		}
		escrow := Escrow{
			Metadata:    &weave.Metadata{Schema: 1},
			Source:      e.Source,
			Arbiter:     e.Arbiter,
			Destination: e.Destination,
			Timeout:     e.Timeout,
			Address:     Condition(key).Address(),
		}
		if _, err := bucket.Put(kv, key, &escrow); err != nil {
			return errors.Wrap(err, "cannot save escrow")
		}
		for _, c := range e.Amount {
			if err := i.Minter.CoinMint(kv, escrow.Address, *c); err != nil {
				return errors.Wrap(err, "failed to issue coins")
			}
		}
	}
	return nil
}
