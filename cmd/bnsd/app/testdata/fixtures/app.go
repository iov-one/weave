package fixtures

import (
	"fmt"
	"math/rand"

	"github.com/iov-one/weave"
	weaveApp "github.com/iov-one/weave/app"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/currency"
	"github.com/iov-one/weave/x/nft/blockchain"
	"github.com/iov-one/weave/x/nft/ticker"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
)

const appState = `
  {
    "cash": [
      {
        "name": "demote",
        "address": "%s",
        "coins": [
          {"whole": 50000, "ticker": "ETH"},
          {"whole": 1234, "ticker": "FRNK"}
        ]
      }
    ]
  }
`

type AppFixture struct {
	Name              string
	ChainID           string
	GenesisKey        *crypto.PrivateKey
	GenesisKeyAddress weave.Address
}

func NewApp() *AppFixture {
	pk := crypto.GenPrivKeyEd25519()
	addr := pk.PublicKey().Address()
	name := fmt.Sprintf("test-%d", rand.Intn(99999999)) //chain id max 20 chars
	return &AppFixture{
		Name:              name,
		ChainID:           fmt.Sprintf("chain-%s", name),
		GenesisKey:        pk,
		GenesisKeyAddress: addr,
	}
}

func (f AppFixture) Build() weaveApp.BaseApp {
	// setup app
	stack := app.Stack(x.Coin{}, nil)
	myApp, err := app.Application(f.Name, stack, app.TxDecoder, "", true)
	if err != nil {
		panic(err)
	}
	myApp.WithInit(weaveApp.ChainInitializers(
		&cash.Initializer{},
		&currency.Initializer{},
		&blockchain.Initializer{},
		&ticker.Initializer{},
	))
	myApp.WithLogger(log.NewNopLogger())
	// load state

	myApp.InitChain(abci.RequestInitChain{AppStateBytes: []byte(fmt.Sprintf(appState, f.GenesisKeyAddress)), ChainId: f.ChainID})
	header := abci.Header{Height: 1}
	myApp.BeginBlock(abci.RequestBeginBlock{Header: header})
	myApp.EndBlock(abci.RequestEndBlock{})
	cres := myApp.Commit()
	block1 := cres.Data
	// sanity check
	if len(block1) == 0 {
		panic("first block must not be empty")
	}
	return myApp
}
