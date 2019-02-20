package app

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/cmd/bnsd/x/nft/username"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/currency"
	"github.com/iov-one/weave/x/distribution"
	"github.com/iov-one/weave/x/multisig"
	"github.com/iov-one/weave/x/validators"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
)

// GenInitOptions will produce some basic options for one rich
// account, to use for dev mode
//
// You can set
func GenInitOptions(args []string) (json.RawMessage, error) {
	ticker := "IOV"
	if len(args) > 0 {
		ticker = args[0]
		if !x.IsCC(ticker) {
			return nil, fmt.Errorf("Invalid ticker %s", ticker)
		}
	}

	var addr string
	if len(args) > 1 {
		addr = args[1]
	} else {
		// if no address provided, auto-generate one
		// and print out a recovery phrase
		bz, phrase, err := GenerateCoinKey()
		if err != nil {
			return nil, err
		}
		addr = hex.EncodeToString(bz)
		fmt.Println(phrase)
	}

	opts := fmt.Sprintf(`
          {
            "cash": [
              {
                "address": "%s",
                "coins": [
                  {"whole": 123456789, "ticker": "%s"}
                ]
              }
            ],
            "currencies": [],
            "nfts": {
              "blockchains": []
            },
            "multisig": [],
	    "update_validators": {
              "addresses": ["%s"]
	    },
	    "distribution": []
          }
	`, addr, ticker, addr)
	return []byte(opts), nil
}

// GenerateApp is used to create a stub for server/start.go command
func GenerateApp(home string, logger log.Logger, debug bool) (abci.Application, error) {
	// db goes in a subdir, but "" stays "" to use memdb
	var dbPath string
	if home != "" {
		dbPath = filepath.Join(home, "bns.db")
	}

	nftBuckets := map[string]orm.Bucket{
		username.ModelName: username.NewBucket().Bucket,
	}
	stack := Stack(nil, nftBuckets)
	application, err := Application("bnsd", stack, TxDecoder, dbPath, debug)
	if err != nil {
		return nil, err
	}
	return DecorateApp(application, logger), nil
}

// DecorateApp adds initializers and Logger to an Application
func DecorateApp(application app.BaseApp, logger log.Logger) app.BaseApp {
	application.WithInit(app.ChainInitializers(
		&gconf.Initializer{},
		&multisig.Initializer{},
		&cash.Initializer{},
		&currency.Initializer{},
		&validators.Initializer{},
		&distribution.Initializer{},
	))
	application.WithLogger(logger)
	return application
}

// InlineApp will take a previously prepared CommitStore and return a complete Application
func InlineApp(kv weave.CommitKVStore, logger log.Logger, debug bool) abci.Application {
	nftBuckets := map[string]orm.Bucket{
		username.ModelName: username.NewBucket().Bucket,
	}
	stack := Stack(nil, nftBuckets)
	ctx := context.Background()
	RegisterNft()
	store := app.NewStoreApp("bnsd", kv, QueryRouter(), ctx)
	base := app.NewBaseApp(store, TxDecoder, stack, nil, debug)
	return DecorateApp(base, logger)
}

type output struct {
	Pubkey *crypto.PublicKey  `json:"pub_key"`
	Secret *crypto.PrivateKey `json:"secret"`
}

// GenerateCoinKey returns the address of a public key,
// along with a json representation of the keys.
// You can give coins to this address and
// import the keys in the js client to use them
func GenerateCoinKey() (weave.Address, string, error) {
	// XXX: we need to generate BIP39 recovery phrases in crypto
	privKey := crypto.GenPrivKeyEd25519()
	pubKey := privKey.PublicKey()
	addr := pubKey.Address()

	out := output{Pubkey: pubKey, Secret: privKey}
	keys, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, "", err
	}

	return addr, string(keys), nil
}
