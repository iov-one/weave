package escrow

import (
	"encoding/hex"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/x/cash"
	"github.com/pkg/errors"
)

var _ weave.Initializer = (*Initializer)(nil)
var burnAddress, _ = hex.DecodeString("0000000000000000000000000000000000000000")

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
		Timeout   int64           `json:"timeout"`
		Amount    []*coin.Coin    `json:"amount"`
	}

	if err := opts.ReadOptions("escrow", &escrows); err != nil {
		return err
	}

	bucket := NewBucket()
	for j, e := range escrows {
		escr := Escrow{
			Sender:    e.Sender,
			Arbiter:   e.Arbiter,
			Recipient: e.Recipient,
			Timeout:   e.Timeout,
			Amount:    e.Amount,
		}
		if err := escr.Validate(); err != nil {
			return errors.Wrapf(err, "invalid escrow at position: %d ", j)
		}
		if !weave.Address(escr.Sender).Equals(burnAddress) {
			// prevent any other address to not generate new money for an existing account (on timeout)
			return errors.New("genesis escrows must have burn address sender")
		}
		obj := bucket.Build(db, &escr)
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
