package fixtures

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/coin"
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
	myApp, err := app.GenerateApp("", log.NewNopLogger(), true)
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
					dict{
						"whole":  50000,
						"ticker": "ETH",
					},
					dict{
						"whole":  1234,
						"ticker": "FRNK",
					},
				},
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
				"amount": []interface{}{
					dict{
						"whole":  1000000,
						"ticker": "FRNK",
					}},
				"timeout": time.Now().Add(10000 * time.Hour),
			},
		},
		"gconf": map[string]interface{}{
			cash.GconfCollectorAddress: "cond:dist/revenue/0000000000000001",
			cash.GconfMinimalFee:       coin.Coin{Ticker: "FRNK", Whole: 0, Fractional: 010000000},
		},
		"msgfee": []interface{}{
			dict{
				"msg_path": "distribution/newrevenue",
				"fee":      coin.Coin{Ticker: "FRNK", Whole: 2},
			},
			dict{
				"msg_path": "distribution/distribute",
				"fee":      coin.Coin{Ticker: "FRNK", Whole: 0, Fractional: 200000000},
			},
			dict{
				"msg_path": "distribution/resetRevenue",
				"fee":      coin.Coin{Ticker: "FRNK", Whole: 1},
			},
			dict{
				"msg_path": "nft/username/issue",
				"fee":      coin.Coin{Ticker: "FRNK", Whole: 5},
			},
		},
	}, "", "  ")
	if err != nil {
		panic(err)
	}
	return appState
}
