package fixtures

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/iov-one/weave"
	weaveApp "github.com/iov-one/weave/app"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/x/nft/blockchain"
	"github.com/iov-one/weave/cmd/bnsd/x/nft/ticker"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/currency"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
)

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
	stack := app.Stack(nil)
	myApp, err := app.Application(f.Name, stack, app.TxDecoder, "", true)
	if err != nil {
		panic(err)
	}
	myApp.WithInit(weaveApp.ChainInitializers(
		&gconf.Initializer{},
		&cash.Initializer{},
		&currency.Initializer{},
		&blockchain.Initializer{},
		&ticker.Initializer{},
	))
	myApp.WithLogger(log.NewNopLogger())
	// load state

	myApp.InitChain(abci.RequestInitChain{
		AppStateBytes: appStateGenesis(f.GenesisKeyAddress),
		ChainId:       f.ChainID,
	})
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

func appStateGenesis(keyAddress weave.Address) []byte {
	type wallet struct {
		Address weave.Address `json:"address"`
		Coins   x.Coins       `json:"coins"`
	}

	state := struct {
		Cash  []wallet               `json:"cash"`
		Gconf map[string]interface{} `json:"gconf"`
	}{
		Cash: []wallet{
			{
				Address: keyAddress,
				Coins: x.Coins{
					{Whole: 50000, Ticker: "ETH"},
					{Whole: 1234, Ticker: "FRNK"},
				},
			},
		},
		Gconf: map[string]interface{}{
			cash.GconfCollectorAddress: "fake-collector-address",
			cash.GconfMinimalFee:       x.Coin{Whole: 0}, // no fee
		},
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		panic(err)
	}
	return raw
}
