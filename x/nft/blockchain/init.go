package blockchain

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
		Blockchains []genesisBlockchainToken `json:"blockchains"`
	}
	if err := opts.ReadOptions("nfts", &nfts); err != nil {
		return errors.Wrap(err, "read options")
	}
	return setTokens(db, nfts.Blockchains)
}

type genesisBlockchainToken struct {
	ID    string        `json:"id"`
	Owner weave.Address `json:"owner"`
	Chain struct {
		ChainID      string `json:"chain_id"`
		NetworkID    string `json:"network_id"`
		Name         string `json:"name"`
		Enabled      bool   `json:"enabled"`
		Production   bool   `json:"production"`
		MainTickerID string `json:"main_ticker_id"`
	} `json:"chain"`
	IOV struct {
		Codec       string `json:"codec"`
		CodecConfig string `json:"codec_config"`
	} `json:"iov"`
}

func setTokens(db weave.KVStore, tokens []genesisBlockchainToken) error {
	bucket := NewBucket()
	for _, t := range tokens {
		chain := Chain{
			ChainID:      t.Chain.ChainID,
			NetworkID:    t.Chain.NetworkID,
			Name:         t.Chain.Name,
			Enabled:      t.Chain.Enabled,
			Production:   t.Chain.Production,
			MainTickerID: []byte(t.Chain.MainTickerID),
		}

		iov := IOV{
			Codec:       t.IOV.Codec,
			CodecConfig: t.IOV.CodecConfig,
		}
		obj, err := bucket.Create(db, t.Owner, []byte(t.ID), nil, chain, iov)
		if err != nil {
			return err
		}
		if err := bucket.Save(db, obj); err != nil {
			return err
		}
	}
	return nil
}
