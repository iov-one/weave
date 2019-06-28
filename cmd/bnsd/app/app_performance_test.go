package bnsd

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/commands/server"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/sigs"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
)

func BenchmarkBnsdEmptyBlock(b *testing.B) {
	var aliceAddr = weavetest.NewKey().PublicKey().Address()

	type dict map[string]interface{}
	genesis := dict{
		"conf": dict{
			"cash": cash.Configuration{
				CollectorAddress: aliceAddr,
				MinimalFee:       coin.Coin{}, // no fee
			},
		},
	}

	bnsd, cleanup := newBnsd(b)
	defer func() {
		b.StopTimer()
		cleanup()
	}()
	runner := weavetest.NewWeaveRunner(b, bnsd, "mychain")
	runner.InitChain(genesis)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		changed := runner.InBlock(func(weavetest.WeaveApp) error {
			// Without sleep this test is locking the CPU.
			time.Sleep(time.Microsecond * 300)
			return nil
		})
		if changed {
			b.Fatal("unexpected change state")
		}
	}
}

func BenchmarkBNSDSendToken(b *testing.B) {
	var (
		aliceKey = weavetest.NewKey()
		alice    = aliceKey.PublicKey().Address()
		benny    = weavetest.NewCondition().Address()
		carol    = weavetest.NewCondition().Address()
	)

	type dict map[string]interface{}
	makeGenesis := func(fee coin.Coin) dict {
		return dict{
			"cash": []interface{}{
				dict{
					"address": alice,
					"coins": []interface{}{
						dict{
							"whole":  123456789,
							"ticker": "IOV",
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
			"conf": dict{
				"cash": cash.Configuration{
					CollectorAddress: carol,
					MinimalFee:       fee,
				},
			},
		}
	}

	cases := map[string]struct {
		txPerBlock int
		fee        coin.Coin
		strategy   weavetest.Strategy
	}{
		"1 tx block, no fee": {
			txPerBlock: 1,
			fee:        coin.Coin{},
			strategy:   weavetest.ExecCheckAndDeliver,
		},
		"1 tx block, no fee (deliver only)": {
			txPerBlock: 1,
			fee:        coin.Coin{},
			strategy:   weavetest.ExecDeliver,
		},
		"10 tx block, no fee": {
			txPerBlock: 10,
			fee:        coin.Coin{},
			strategy:   weavetest.ExecCheckAndDeliver,
		},
		"100 tx block, no fee": {
			txPerBlock: 100,
			fee:        coin.Coin{},
			strategy:   weavetest.ExecCheckAndDeliver,
		},
		"100 tx block with fee": {
			txPerBlock: 100,
			fee:        coin.Coin{Whole: 1, Ticker: "IOV"},
			strategy:   weavetest.ExecCheckAndDeliver,
		},
		"100 tx block with fee, check only": {
			txPerBlock: 100,
			fee:        coin.Coin{Whole: 1, Ticker: "IOV"},
			strategy:   weavetest.ExecCheck,
		},
		"100 tx block with fee, deliver only": {
			txPerBlock: 100,
			fee:        coin.Coin{Whole: 1, Ticker: "IOV"},
			strategy:   weavetest.ExecDeliver,
		},
		"100 tx block with fee, deliver with precheck": {
			txPerBlock: 100,
			fee:        coin.Coin{Whole: 1, Ticker: "IOV"},
			strategy:   weavetest.ExecCheckAndDeliver | weavetest.NoBenchCheck,
		},
	}

	for testName, tc := range cases {
		b.Run(testName, func(b *testing.B) {
			bnsd, cleanup := newBnsd(b)
			defer func() {
				b.StopTimer()
				cleanup()
			}()
			runner := weavetest.NewWeaveRunner(b, bnsd, "mychain")
			runner.InitChain(makeGenesis(tc.fee))

			// We are the only user of this bnsd instance so we can
			// easily predict the nonce for alice. No need to ask
			// the database.
			var aliceNonce int64

			var fees *cash.FeeInfo
			if !tc.fee.IsZero() {
				fees = &cash.FeeInfo{
					Payer: alice,
					Fees:  &tc.fee,
				}
			}

			// Generate all transactions before measuring.
			txs := make([]weave.Tx, b.N)
			for k := 0; k < b.N; k++ {
				tx := &Tx{
					Fees: fees,
					Sum: &Tx_CashSendMsg{
						CashSendMsg: &cash.SendMsg{
							Source:      alice,
							Destination: benny,
							Amount:      coin.NewCoinp(0, 100, "IOV"),
						},
					},
				}
				sig, err := sigs.SignTx(aliceKey, tx, "mychain", aliceNonce)
				if err != nil {
					b.Fatalf("cannot sign transaction %+v", err)
				}
				tx.Signatures = append(tx.Signatures, sig)
				txs[k] = tx

				if !tc.strategy.Has(weavetest.ExecDeliver) && (k+1)%tc.txPerBlock == 0 {
					// When the transaction is split into blocks and the previous block
					// is not committed then the nonce value is reset. All previous
					// changes were discarded so the nonce counting starts from zero again.
					aliceNonce = 0
				} else {
					aliceNonce++
				}
			}

			blocks := weavetest.SplitTxs(txs, tc.txPerBlock)

			b.ResetTimer()
			runner.ProcessAllTxs(blocks, tc.strategy)
		})
	}
}

// newBnsd returns the test application, along with a function to delete all
// testdata at the end.
func newBnsd(t testing.TB) (abci.Application, func()) {
	t.Helper()

	homeDir, err := ioutil.TempDir("", "bnsd_performance_home")
	if err != nil {
		t.Fatalf("cannot create a temporary directory: %s", err)
	}
	opts := &server.Options{
		MinFee: coin.Coin{},
		Home:   homeDir,
		Logger: log.NewNopLogger(),
		Debug:  false,
	}
	bnsd, err := GenerateApp(opts)
	if err != nil {
		t.Fatalf("cannot generate bnsd instance: %s", err)
	}

	cleanup := func() {
		os.RemoveAll(homeDir)
	}
	return bnsd, cleanup
}
