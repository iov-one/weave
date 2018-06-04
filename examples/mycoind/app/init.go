package app

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/tmlibs/log"

	"github.com/confio/weave"
	"github.com/confio/weave/crypto"
	"github.com/confio/weave/x"
	"github.com/confio/weave/x/cash"
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

	opts := fmt.Sprintf(`{
    "cash": [
      {
        "address": "%s",
        "coins": [
          {
            "whole": 123456789,
            "ticker": "%s"
          }
        ]
      }
    ]
  }`, addr, code)
	return []byte(opts), nil
}

// GenerateApp is used to create a stub for server/start.go command
func GenerateApp(home string, logger log.Logger) (abci.Application, error) {
	// db goes in a subdir, but "" -> "" for memdb
	var dbPath string
	if home != "" {
		dbPath = filepath.Join(home, "abci.db")
	}

	stack := Stack(x.Coin{})
	app, err := Application("mycoin", stack, TxDecoder, dbPath)
	if err != nil {
		return nil, err
	}
	app.WithInit(cash.Initializer{})

	// set the logger and return
	app.WithLogger(logger)
	return app, nil
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
