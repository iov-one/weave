package scenarios

import (
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
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/commands/server"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/x/distribution"
	"github.com/iov-one/weave/x/escrow"
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
	derivationPath    = flag.String("derivation", "", "bip44 derivation path: \"m/44'/234'/0'\"")
)

var (
	alice             *client.PrivateKey
	node              *nm.Node
	logger            = log.NewTMLogger(os.Stdout)
	bnsClient         *client.BnsClient
	chainID           string
	rpcAddress        string
	multiSigContract  weave.Condition
	escrowContract    weave.Condition
	distrContractAddr weave.Address
	antiSpamFee       = coin.Coin{Ticker: "IOV", Whole: 0, Fractional: 100000000}
)

func TestMain(m *testing.M) {
	flag.Parse()
	multiSigContract = multisig.MultiSigCondition(weavetest.SequenceID(1))
	escrowContract = escrow.Condition(weavetest.SequenceID(1))
	distrContractAddr, _ = distribution.RevenueAccount(weavetest.SequenceID(1))

	alice = derivePrivateKey(*hexSeed, *derivationPath)
	logger.Info("Loaded Alice key", "addressID", alice.PublicKey().Address().String(), "derivationPath", *derivationPath)

	if *tendermintAddress != testLocalAddress {
		bnsClient = client.NewClient(client.NewHTTPConnection(*tendermintAddress))
		var err error
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
	opts := &server.Options{
		MinFee: coin.Coin{},
		Home:   config.RootDir,
		Logger: logger,
		Debug:  false,
	}
	bnsd, err := app.GenerateApp(opts)
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
				"ticker": "IOV",
				"name":   "Main token of this chain",
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
					dict{"power": 1, "signature": addr},
				},
				"activation_threshold": 1,
				"admin_threshold":      1,
			},
		},
		"distribution": []interface{}{
			dict{
				"admin": "cond:multisig/usage/0000000000000001",
				"recipients": []interface{}{
					dict{"weight": 1, "address": alice.PublicKey().Address()},
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
						"ticker": "IOV",
					}},
				"timeout": time.Now().Add(10000 * time.Hour),
			},
		},
		"gconf": app.Configuration{
			CollectorAddress: weave.Condition("cond:dist/revenue/0000000000000001").Address(),
			MinimalFee:       antiSpamFee,
		},
		"msgfee": []interface{}{
			dict{
				"msg_path": "distribution/newrevenue",
				"fee":      coin.Coin{Ticker: "IOV", Whole: 2},
			},
			dict{
				"msg_path": "distribution/distribute",
				"fee":      coin.Coin{Ticker: "IOV", Whole: 0, Fractional: 200000000},
			},
			dict{
				"msg_path": "distribution/resetRevenue",
				"fee":      coin.Coin{Ticker: "IOV", Whole: 1},
			},
			dict{
				"msg_path": "nft/username/issue",
				"fee":      coin.Coin{Ticker: "IOV", Whole: 5},
			},
		},
	}, "", "  ")
	if err != nil {
		panic(err)
	}
	doc.AppState = appState
	// save file
	return doc, doc.SaveAs(filename)
}

// derivePrivateKey derive a private key from hex and given path. Path can be empty to not derive.
func derivePrivateKey(hexSeed, path string) *client.PrivateKey {
	if len(path) != 0 {
		seed, err := hex.DecodeString(hexSeed)
		if err != nil {
			logger.Error("Failed to decode private key", "cause", err)
			os.Exit(1)
		}
		k, err := derivation.DeriveForPath(path, seed)
		if err != nil {
			logger.Error("Failed to derive private key", "cause", err, "path", path)
			os.Exit(1)
		}
		pubKey, err := k.PublicKey()
		if err != nil {
			logger.Error("Failed to derive public key", "cause", err)
			os.Exit(1)
		}
		hexSeed = hex.EncodeToString(append(k.Key, pubKey...))
	}
	pk, err := client.DecodePrivateKeyFromSeed(hexSeed)
	if err != nil {
		logger.Error("Failed to decode private key", "cause", err)
		os.Exit(1)
	}
	return pk
}

// seedAccountWithTokens acts as a faucet that sends tokens to the given address.
func seedAccountWithTokens(dest weave.Address) {
	cc := coin.NewCoin(10, 0, "IOV")
	tx := client.BuildSendTx(alice.PublicKey().Address(), dest, cc, "faucet")
	tx.Fee(alice.PublicKey().Address(), antiSpamFee)

	aNonce := client.NewNonce(bnsClient, alice.PublicKey().Address())
	seq, err := aNonce.Next()
	if err != nil {
		panic(err)
	}
	if err := client.SignTx(tx, alice, chainID, seq); err != nil {
		panic(err)
	}
	resp := bnsClient.BroadcastTx(tx)
	if err := resp.IsError(); err != nil {
		panic(err)
	}
}
