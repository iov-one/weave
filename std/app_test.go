package std

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	abci "github.com/tendermint/abci/types"

	"github.com/confio/weave/crypto"
	"github.com/confio/weave/x/coins"
)

func TestApp(t *testing.T) {
	// no minimum fee, in-memory data-store
	stack := Stack(coins.Coin{})
	app, err := Application("demo", stack, TxDecoder, "")
	require.NoError(t, err)

	// let's set up a genesis file with some cash
	pk := crypto.GenPrivKeyEd25519()
	addr := pk.PublicKey().Address()
	genesis := fmt.Sprintf(`{
        "chain_id": "test-net-22",
        "app_state": {
            "coins": [{
                "address": "%X",
                "coins": [{
                    "integer": 50000,
                    "currency_code": "ETH"
                    }, {
                    "integer": 1234,
                    "currency_code": "FRNK"
                }]
            }]
        }
    }`, addr)
	app.StoreApp.WithInit(coins.Initializer{})

	// Commit first block, make sure non-nil hash
	app.InitChainWithGenesis(abci.RequestInitChain{}, []byte(genesis))
	header := abci.Header{Height: 1}
	app.BeginBlock(abci.RequestBeginBlock{Header: &header})
	app.EndBlock(abci.RequestEndBlock{})
	res := app.Commit()
	assert.NotEmpty(t, res.Data)

	// Query for my balance

	// Send some money

	// Query for new balances

}
