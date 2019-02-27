package app

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
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
		"gconf": dict{
			cash.GconfCollectorAddress: "66616b652d636f6c6c6563746f722d61646472657373",
			cash.GconfMinimalFee:       x.Coin{Whole: 0}, // no fee
		},
	})
}

// GenerateApp is used to create a stub for server/start.go command
func GenerateApp(home string, logger log.Logger, debug bool) (abci.Application, error) {
	// db goes in a subdir, but "" -> "" for memdb
	var dbPath string
	if home != "" {
		dbPath = filepath.Join(home, "abci.db")
	}

	stack := Stack(x.Coin{})
	application, err := Application("mycoin", stack, TxDecoder, dbPath, debug)
	if err != nil {
		return nil, err
	}
	application.WithInit(app.ChainInitializers(
		&gconf.Initializer{},
		&cash.Initializer{},
	))

	// set the logger and return
	application.WithLogger(logger)
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
