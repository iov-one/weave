package fixtures

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/commands/server"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/x/cash"
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

func (f AppFixture) Build() abci.Application {
	opts := &server.Options{
		MinFee: coin.Coin{},
		Home:   "",
		Logger: log.NewNopLogger(),
		Debug:  true,
	}
	myApp, err := app.GenerateApp(opts)
	if err != nil {
		panic(err)
	}

	// setup app
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
	type dict map[string]interface{}

	appState, err := json.MarshalIndent(dict{
		"cash": []interface{}{
			dict{
				"address": keyAddress,
				"coins": []interface{}{
					"50000 ETH", "1234 FRNK",
				},
			},
		},
		"conf": dict{
			"cash": cash.Configuration{
				CollectorAddress: weave.Condition("dist/revenue/0000000000000001").Address(),
				MinimalFee:       coin.NewCoin(0, 10000000, "FRNK"),
			},
		},
		"currencies": []interface{}{
			dict{
				"ticker": "FRNK",
				"name":   "Utility token of this chain",
			},
			dict{
				"ticker": "ETH",
				"name":   "Other token of this chain",
			},
		},
		"update_validators": dict{
			"addresses": []interface{}{
				"cond:multisig/usage/0000000000000001",
			},
		},
		"multisig": []interface{}{
			dict{
				"participants": []interface{}{
					dict{"power": 1, "signature": keyAddress},
				},
				"activation_threshold": 1,
				"admin_threshold":      1,
			},
		},
		"distribution": []interface{}{
			dict{
				"admin": "cond:multisig/usage/0000000000000001",
				"recipients": []interface{}{
					dict{"weight": 1, "address": keyAddress},
				},
			},
		},
		"escrow": []interface{}{
			dict{
				"sender":    "0000000000000000000000000000000000000000",
				"arbiter":   "multisig/usage/0000000000000001",
				"recipient": "cond:dist/revenue/0000000000000001",
				"amount":    []interface{}{"1000000 FRNK"},
				"timeout":   time.Now().Add(10000 * time.Hour),
			},
		},
		"msgfee": []interface{}{
			dict{
				"msg_path": "distribution/newrevenue",
				"fee":      "2 FRNK",
			},
			dict{
				"msg_path": "distribution/distribute",
				"fee":      "0.2FRNK",
			},
			dict{
				"msg_path": "distribution/resetRevenue",
				"fee":      "1 FRNK",
			},
			dict{
				"msg_path": "nft/username/issue",
				"fee":      "5 FRNK",
			},
		},
	}, "", "  ")
	if err != nil {
		panic(err)
	}
	return appState
}
