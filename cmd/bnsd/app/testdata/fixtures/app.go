package fixtures

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/commands/server"
	"github.com/iov-one/weave/crypto"
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
	myApp, err := bnsd.GenerateApp(opts)
	if err != nil {
		panic(err)
	}

	// setup app
	myApp.InitChain(abci.RequestInitChain{
		AppStateBytes: appStateGenesis(f.GenesisKeyAddress),
		ChainId:       f.ChainID,
	})
	header := abci.Header{
		Height: 1,
		Time:   time.Now(),
	}
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

	state := dict{
		"cash": []interface{}{
			dict{
				"address": keyAddress,
				"coins": []interface{}{
					"50000 ETH", "1234 FRNK",
				},
			},
		},
		"conf": dict{
			"cash": dict{
				"collector_address": "seq:dist/revenue/1",
				"minimal_fee":       "0.01 FRNK",
			},
			"migration": dict{
				"admin": "seq:multisig/usage/1",
			},
		},
		"initialize_schema": []dict{
			{"ver": 1, "pkg": "batch"},
			{"ver": 1, "pkg": "cash"},
			{"ver": 1, "pkg": "cron"},
			{"ver": 1, "pkg": "currency"},
			{"ver": 1, "pkg": "distribution"},
			{"ver": 1, "pkg": "escrow"},
			{"ver": 1, "pkg": "gov"},
			{"ver": 1, "pkg": "msgfee"},
			{"ver": 1, "pkg": "multisig"},
			{"ver": 1, "pkg": "paychan"},
			{"ver": 1, "pkg": "sigs"},
			{"ver": 1, "pkg": "username"},
			{"ver": 1, "pkg": "utils"},
			{"ver": 1, "pkg": "validators"},
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
				"seq:multisig/usage/1",
			},
		},
		"multisig": []interface{}{
			dict{
				"participants": []interface{}{
					dict{"weight": 1, "signature": keyAddress},
				},
				"activation_threshold": 1,
				"admin_threshold":      1,
			},
		},
		"distribution": []interface{}{
			dict{
				"admin": "seq:multisig/usage/1",
				"destinations": []interface{}{
					dict{"weight": 1, "address": keyAddress},
				},
			},
		},
		"escrow": []interface{}{
			dict{
				"source":      "0000000000000000000000000000000000000000",
				"arbiter":     "seq:multisig/usage/1",
				"destination": "seq:dist/revenue/1",
				"amount":      []interface{}{"1000000 FRNK"},
				"timeout":     time.Now().Add(10000 * time.Hour),
			},
		},
		"msgfee": []interface{}{
			dict{
				"msg_path": "distribution/create",
				"fee":      "2 FRNK",
			},
			dict{
				"msg_path": "distribution/distribute",
				"fee":      "0.2FRNK",
			},
			dict{
				"msg_path": "distribution/reset",
				"fee":      "1 FRNK",
			},
			dict{
				"msg_path": "username/register_token",
				"fee":      "5 FRNK",
			},
		},
	}
	appState, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		panic(err)
	}
	return appState
}
