package std

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/tmlibs/log"

	"github.com/confio/weave/x/coins"
)

// GenInitOptions will produce some basic options for one rich
// account, to use for dev mode
func GenInitOptions(args []string) (json.RawMessage, error) {
	// TODO: make these configurable
	code := "MYC"
	addr := "0102030405060708090021222324252627282930"

	opts := fmt.Sprintf(`{
    "accounts": [
      {
        "address": "%s",
        "coins": [
          {
            "integer": 123456789,
            "currency_code": "%s"
          }
        ]
      }
    ]
  }`, addr, code)
	return []byte(opts), nil
}

// GenerateApp is used to create a stub for server/start.go command
func GenerateApp(dbPath string, logger log.Logger) (abci.Application, error) {
	stack := Stack(coins.Coin{})
	app, err := Application("mycoin", stack, TxDecoder, dbPath)
	if err != nil {
		return nil, err
	}
	app.WithInit(coins.Initializer{})

	// guess the location of the genesis file
	genesisPath := filepath.Join(dbPath, "config", "genesis.json")
	app.WithGenesis(genesisPath)

	// set the logger and return
	app.WithLogger(logger)
	return app, nil
}
