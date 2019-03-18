package app

import (
	"encoding/hex"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
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
		"gconf": map[string]interface{}{
			cash.GconfCollectorAddress: hex.EncodeToString(aliceAddr),
			cash.GconfMinimalFee:       coin.Coin{}, // no fee
		},
	}

	bnsd := newBnsd(b)
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
		alice = weavetest.NewKey()
		benny = weavetest.NewKey()
		carol = weavetest.NewKey()
	)

	type dict map[string]interface{}
	makeGenesis := func(fee coin.Coin) dict {
		return dict{
			"cash": []interface{}{
				dict{
					"address": alice.PublicKey().Address(),
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
			"gconf": dict{
				cash.GconfCollectorAddress: hex.EncodeToString(carol.PublicKey().Address()),
				cash.GconfMinimalFee:       fee,
			},
		}
	}

	cases := map[string]struct {
		txPerBlock int
		fee        coin.Coin
	}{
		"1 tx, no fee": {
			txPerBlock: 1,
			fee:        coin.Coin{},
		},
		"15 tx, no fee": {
			txPerBlock: 15,
			fee:        coin.Coin{},
		},
	}

	for testName, tc := range cases {
		b.Run(testName, func(b *testing.B) {
			bnsd := newBnsd(b)
			runner := weavetest.NewWeaveRunner(b, bnsd, "mychain")
			runner.InitChain(makeGenesis(tc.fee))

			aliceNonce := NewNonce(runner, alice.PublicKey().Condition().Address())

			b.ResetTimer()

			runs := b.N / tc.txPerBlock
			rem := b.N % tc.txPerBlock
			if rem > 0 {
				runs++
			}

			// b.Logf("Testcase with %d txs", b.N)
			for i := 0; i < runs; i++ {
				changed := runner.InBlock(func(wapp weavetest.WeaveApp) error {
					numTxs := tc.txPerBlock
					if i == runs-1 && rem > 0 {
						numTxs = rem
					}
					// b.Logf("Running block with %d tx", numTxs)

					for j := 0; j < numTxs; j++ {

						tx := Tx{
							// TODO: select fee
							Sum: &Tx_SendMsg{
								&cash.SendMsg{
									Src:    alice.PublicKey().Address(),
									Dest:   benny.PublicKey().Address(),
									Amount: coin.NewCoinp(0, 100, "IOV"),
								},
							},
						}

						nonce, err := aliceNonce.Next()
						if err != nil {
							return err
						}

						sig, err := sigs.SignTx(alice, &tx, "mychain", nonce)
						if err != nil {
							return errors.Wrap(err, "cannot sign transaction")
						}
						tx.Signatures = append(tx.Signatures, sig)

						if err := wapp.CheckTx(&tx); err != nil {
							return errors.Wrap(err, "cannot check tx")
						}
						if err := wapp.DeliverTx(&tx); err != nil {
							return errors.Wrap(err, "cannot deliver tx")
						}
					}
					return nil
				})
				if !changed {
					b.Fatal("unexpected change state")
				}
			}
		})
	}
}

func newBnsd(t weavetest.Tester) abci.Application {
	t.Helper()

	homeDir, err := ioutil.TempDir("", "bnsd_performance_home")
	if err != nil {
		t.Fatalf("cannot create a temporary directory: %s", err)
	}
	bnsd, err := GenerateApp(homeDir, log.NewNopLogger(), false)
	if err != nil {
		t.Fatalf("cannot generate bnsd instance: %s", err)
	}
	return bnsd
}

// Nonce has a client/address pair, queries for the nonce
// and caches recent nonce locally to quickly sign
type Nonce struct {
	mutex     sync.Mutex
	db        weave.ReadOnlyKVStore
	bucket    sigs.Bucket
	addr      weave.Address
	nonce     int64
	fromQuery bool
}

// NewNonce creates a nonce for a client / address pair.
// Call Query to force a query, Next to use cache if possible
func NewNonce(db weave.ReadOnlyKVStore, addr weave.Address) *Nonce {
	return &Nonce{
		db:     db,
		addr:   addr,
		bucket: sigs.NewBucket(),
	}
}

// Query always queries the blockchain for the next nonce
func (n *Nonce) Query() (int64, error) {
	obj, err := n.bucket.Get(n.db, n.addr)
	if err != nil {
		return 0, err
	}
	user := sigs.AsUser(obj)

	n.mutex.Lock()
	if user == nil { // Nonce not found
		n.nonce = 0
	} else {
		n.nonce = user.Sequence
	}
	n.fromQuery = true
	n.mutex.Unlock()
	return n.nonce, nil
}

// Next will use a cached value if present, otherwise Query
// It will always increment by 1, assuming last nonce
// was properly used. This is designed for cases where
// you want to rapidly generate many tranasactions without
// querying the blockchain each time
func (n *Nonce) Next() (int64, error) {
	n.mutex.Lock()
	initializeFromBlockchain := !n.fromQuery && n.nonce == 0
	n.mutex.Unlock()
	if initializeFromBlockchain {
		return n.Query()
	}
	n.mutex.Lock()
	n.nonce++
	n.fromQuery = false
	result := n.nonce
	n.mutex.Unlock()
	return result, nil
}
