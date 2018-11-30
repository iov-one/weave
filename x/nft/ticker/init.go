package ticker

import (
	"github.com/iov-one/weave"
	"github.com/pkg/errors"
)

// Initializer fulfils the InitStater interface to load data from
// the genesis file
type Initializer struct {
}

var _ weave.Initializer = (*Initializer)(nil)

// FromGenesis will parse initial tokens information from the genesis and
// persist it in the database.
func (i *Initializer) FromGenesis(opts weave.Options, db weave.KVStore) error {
	var nfts struct {
		Tickers []genesisTickerToken `json:"tickers"`
	}
	if err := opts.ReadOptions("nfts", &nfts); err != nil {
		return errors.Wrap(err, "read options")
	}
	return setTickerTokens(db, nfts.Tickers)
}

// keep it close to the TickerToken model definition
type genesisTickerToken struct {
	Details struct {
		BlockchainID string `json:"blockchain_id"`
	} `json:"details"`
	Base struct {
		ID    string `json:"id"`
		Owner string `json:"owner"`
	} `json:"base"`
}

func setTickerTokens(db weave.KVStore, tokens []genesisTickerToken) error {
	bucket := NewBucket()
	for _, t := range tokens {
		obj, err := bucket.Create(db, weave.Address(t.Base.Owner), weave.Address(t.Base.ID), nil, []byte(t.Details.BlockchainID))
		if err != nil {
			return err
		}
		if err := bucket.Save(db, obj); err != nil {
			return err
		}
	}
	return nil
}
