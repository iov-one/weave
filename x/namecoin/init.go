package namecoin

import (
	"encoding/json"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

const (
	optWallet = "wallets"
	optToken  = "tokens"
)

// GenesisAccount is used to parse the json from genesis file
// use weave.Address, so address in hex, not base64
type GenesisAccount struct {
	Address weave.Address `json:"address"`
	*Wallet
}

// GenesisToken is used to describe a token in the genesis account
type GenesisToken struct {
	Ticker  string `json:"ticker"`
	Name    string `json:"name"`
	SigFigs int32  `json:"sig_figs"`
}

// ToGenesisToken converts internal structs to genesis file format
func ToGenesisToken(ticker string, token *Token) GenesisToken {
	return GenesisToken{
		Ticker:  ticker,
		Name:    token.GetName(),
		SigFigs: token.GetSigFigs(),
	}
}

// Initializer fulfils the InitStater interface to load data from
// the genesis file
type Initializer struct{}

var _ weave.Initializer = Initializer{}

// FromGenesis will parse initial account info from genesis
// and save it to the database
func (Initializer) FromGenesis(opts weave.Options, db weave.KVStore) error {
	accts := []GenesisAccount{}
	err := opts.ReadOptions(optWallet, &accts)
	if err != nil {
		return err
	}
	err = setWallets(db, accts)
	if err != nil {
		return err
	}

	tokens := []GenesisToken{}
	err = opts.ReadOptions(optToken, &tokens)
	if err != nil {
		return err
	}
	err = setTokens(db, tokens)
	if err != nil {
		return err
	}
	return nil
}

func setWallets(db weave.KVStore, gens []GenesisAccount) error {
	bucket := NewWalletBucket()
	for _, gen := range gens {
		if len(gen.Address) != weave.AddressLength {
			return errors.ErrInvalidInput.Newf("address: %v", gen.Address)
		}
		wallet, err := WalletWith(gen.Address, gen.Name, gen.Wallet.Coins...)
		if err != nil {
			return err
		}
		err = bucket.Save(db, wallet)
		if err != nil {
			return err
		}
	}
	return nil
}

func setTokens(db weave.KVStore, gens []GenesisToken) error {
	bucket := NewTokenBucket()
	for _, gen := range gens {
		token := NewToken(gen.Ticker, gen.Name, gen.SigFigs)
		err := bucket.Save(db, token)
		if err != nil {
			return err
		}
	}
	return nil
}

// BuildGenesis will create Options with the given the wallets and tokens
func BuildGenesis(wallets []GenesisAccount,
	tokens []GenesisToken) (weave.Options, error) {

	opts := make(weave.Options, 2)

	if len(wallets) > 0 {
		walletBz, err := json.MarshalIndent(wallets, "", "  ")
		if err != nil {
			return nil, err
		}
		opts[optWallet] = walletBz
	}

	if len(tokens) > 0 {
		tokenBz, err := json.MarshalIndent(tokens, "", "  ")
		if err != nil {
			return nil, err
		}
		opts[optToken] = tokenBz
	}

	return opts, nil
}
