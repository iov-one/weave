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
func (i *Initializer) FromGenesis(opts weave.Options, db weave.KVStore) error {
	var escrows []struct {
		Sender    weave.Address   `json:"sender"`
		Arbiter   weave.Condition `json:"arbiter"`
		Recipient weave.Address   `json:"recipient"`
		Timeout   weave.UnixTime  `json:"timeout"`
		Amount    []*coin.Coin    `json:"amount"`
	}

	if err := opts.ReadOptions("escrow", &escrows); err != nil {
		return err
	}

	bucket := NewBucket()
	for j, e := range escrows {
		escr := Escrow{
			Metadata:  &weave.Metadata{Schema: 1},
			Sender:    e.Sender,
			Arbiter:   e.Arbiter,
			Recipient: e.Recipient,
			Timeout:   e.Timeout,
		}
		if err := escr.Validate(); err != nil {
			return errors.Wrapf(err, "invalid escrow at position: %d ", j)
		}
		obj, err := bucket.Build(db, &escr)
		if err != nil {
			return err
		}
		if err := bucket.Save(db, obj); err != nil {
			return err
		}
		escAddr := Condition(obj.Key()).Address()
		for _, c := range e.Amount {
			if err := i.Minter.CoinMint(db, escAddr, *c); err != nil {
				return errors.Wrap(err, "failed to issue coins")
			}
		}
	}
	return nil
}
