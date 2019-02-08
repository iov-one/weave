package scenarios

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/multisig"
	"github.com/stellar/go/exp/crypto/derivation"
	abci "github.com/tendermint/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/rpc/test"
	tm "github.com/tendermint/tendermint/types"
)

// application version, will be set during compilation time

const (
	startupDelay     = 100 * time.Millisecond // for tendermint setup
	testLocalAddress = "localhost:46657"
)

var (
	tendermintAddress = flag.String("address", testLocalAddress, "destination address of tendermint rpc")
	hexSeed           = flag.String("seed", "d34c1970ae90acf3405f2d99dcaca16d0c7db379f4beafcfdf667b9d69ce350d27f5fb440509dfa79ec883a0510bc9a9614c3d44188881f0c5e402898b4bf3c9", "private key seed in hex")
	delay             = flag.Duration("delay", time.Duration(0), "duration to wait between test cases for rate limits")
	derivationPath    = flag.String("derivation", "", "bip44 derivation path: \"m/4804438'/0'\"")
)

var (
	alice                *client.PrivateKey
	node                 *nm.Node
	logger               = log.NewTMLogger(os.Stdout) //log.NewNopLogger()
	bnsClient            *client.BnsClient
	chainID              string
	rpcAddress           string
	multiSigContractID   = make([]byte, 8) // first contractID
	multiSigContractAddr weave.Address     // results to: "5AE2C58796B0AD48FFE7602EAC3353488C859A2B"
)

func TestMain(m *testing.M) {
	flag.Parse()
	binary.BigEndian.PutUint64(multiSigContractID, 1)
	multiSigContractAddr = multisig.MultiSigCondition(multiSigContractID).Address()
	var err error
	aliceHexSeed := *hexSeed
	if len(*derivationPath) != 0 {
		b, err := hex.DecodeString(*hexSeed)
		if err != nil {
			logger.Error("Failed to decode private key", "cause", err)
			os.Exit(1)
		}
		k, err := derivation.DeriveForPath(*derivationPath, b)
		if err != nil {
			logger.Error("Failed to derive private key", "cause", err, "derivationPath", *derivationPath)
			os.Exit(1)
		}
		pubKey, err := k.PublicKey()
		if err != nil {
			logger.Error("Failed to derive public key", "cause", err)
			os.Exit(1)
		}
		aliceHexSeed = hex.EncodeToString(append(k.Key, pubKey...))
	}

	alice, err = client.DecodePrivateKeyFromSeed(aliceHexSeed)
	if err != nil {
		logger.Error("Failed to decode private key", "cause", err)
		os.Exit(1)
	}
	logger.Error("Loaded Alice key", "addressID", alice.PublicKey().Address())

	if *tendermintAddress != testLocalAddress {
		bnsClient = client.NewClient(client.NewHTTPConnection(*tendermintAddress))
		chainID, err = bnsClient.ChainID()
		if err != nil {
			logger.Error("Failed to fetch chain id", "cause", err)
			os.Exit(1)
		}
		rpcAddress = *tendermintAddress
		os.Exit(m.Run())
	}

	config := rpctest.GetConfig()
	config.Moniker = "SetInTestMain"
	chainID = config.ChainID()

	rpcAddress = "http://localhost" + config.RPC.ListenAddress[strings.LastIndex(config.RPC.ListenAddress, ":"):]
	app, err := initApp(config, alice.PublicKey().Address())
	if err != nil {
		logger.Error("Failed to init app", "cause", err)
		os.Exit(1)
	}

	// run the app inside a tendermint instance
	node = rpctest.StartTendermint(app)
	bnsClient = client.NewClient(client.NewLocalConnection(node))
	time.Sleep(startupDelay) // wait for chain
	code := m.Run()

	// and shut down proper at the end
	node.Stop()
	node.Wait()
	os.Exit(code)
}

func delayForRateLimits() {
	time.Sleep(*delay)
}

func initApp(config *cfg.Config, addr weave.Address) (abci.Application, error) {
	bnsd, err := app.GenerateApp(config.RootDir, logger, false)
	if err != nil {
		return nil, err
	}

	// generate genesis file...
	_, err = initGenesis(config.GenesisFile(), addr)
	return bnsd, err
}

func initGenesis(filename string, addr weave.Address) (*tm.GenesisDoc, error) {
	// load genesis
	doc, err := tm.GenesisDocFromFile(filename)
	if err != nil {
		return nil, err
	}

	type dict map[string]interface{}

	appState, err := json.MarshalIndent(dict{
		"cash": []interface{}{
			dict{
				"address": addr,
				"coins": []interface{}{
					dict{
						"whole":  123456789,
						"ticker": "IOV",
					},
					dict{
						"whole":  123456789,
						"ticker": "CASH",
					},
					dict{
						"whole":  123456789,
						"ticker": "ALX",
					},
					dict{
						"whole":  123456789,
						"ticker": "PAJA",
					},
				},
			},
		},
		"currencies": []interface{}{
			dict{
				"ticker":   "IOV",
				"name":     "Main token of this chain",
				"sig_figs": 6,
			},
		},
		"update_validators": dict{
			"addresses": []weave.Address{
				multiSigContractAddr,
			},
		},
		"multisig": []interface{}{
			dict{
				"sigs":                 []weave.Address{addr},
				"activation_threshold": 1,
				"admin_threshold":      1,
			},
		},
		"gconf": map[string]interface{}{
			cash.GconfCollectorAddress: hex.EncodeToString(addr),
			cash.GconfMinimalFee:       x.Coin{}, // no fee
		},
	}, "", "  ")
	if err != nil {
		panic(err)
	}

	doc.AppState = appState
	// save file
	return doc, doc.SaveAs(filename)
}
