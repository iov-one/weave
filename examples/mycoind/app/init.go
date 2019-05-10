package app

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/commands/server"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/validators"
	abci "github.com/tendermint/tendermint/abci/types"
)

// GenInitOptions will produce some basic options for one rich
// account, to use for dev mode
//
// You can set
func GenInitOptions(args []string) (json.RawMessage, error) {
	code := "MYC"
	if len(args) > 0 {
		code = args[0]
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
		addr = bz.String()
		fmt.Println(phrase)
	}

	type (
		dict  map[string]interface{}
		array []interface{}
	)
	collectorAddr, err := hex.DecodeString("3b11c732b8fc1f09beb34031302fe2ab347c5c14")
	if err != nil {
		return nil, errors.Wrap(err, "cannot hex decode collector address")
	}
	return json.Marshal(dict{
		"cash": array{
			dict{
				"address": addr,
				"coins": array{
					dict{
						"whole":  123456789,
						"ticker": code,
					},
				},
			},
		},
		"conf": dict{
			"cash": cash.Configuration{
				CollectorAddress: collectorAddr,
				MinimalFee:       coin.Coin{Whole: 0}, // no fee
			},
			"migration": dict{
				"admin": addr,
			},
		},
		"initialize_schema": []dict{
			{"pkg": "cash", "ver": 1},
			{"pkg": "sigs", "ver": 1},
			{"pkg": "validators", "ver": 1},
			{"pkg": "utils", "ver": 1},
		},
	})
}

// GenerateApp is used to create a stub for server/start.go command
func GenerateApp(options *server.Options) (abci.Application, error) {
	// db goes in a subdir, but "" -> "" for memdb
	var dbPath string
	if options.Home != "" {
		dbPath = filepath.Join(options.Home, "abci.db")
	}

	stack := Stack(options.MinFee)
	application, err := Application("mycoin", stack, TxDecoder, dbPath, options.Debug)
	if err != nil {
		return nil, err
	}
	application.WithInit(app.ChainInitializers(
		&migration.Initializer{},
		&cash.Initializer{},
		&validators.Initializer{},
	))

	// set the logger and return
	application.WithLogger(options.Logger)
	return application, nil
}

// GenerateCoinKey returns the address of a public key,
// along with the secret phrase to recover the private key.
// You can give coins to this address and return the recovery
// phrase to the user to access them.
func GenerateCoinKey() (weave.Address, string, error) {
	// XXX: we need to generate BIP39 recovery phrases in crypto
	privKey := crypto.GenPrivKeyEd25519()
	addr := privKey.PublicKey().Address()
	return addr, "TODO: add a recovery phrase", nil
}
