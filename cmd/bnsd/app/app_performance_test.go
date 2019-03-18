package app

import (
	"encoding/hex"
	"io/ioutil"
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

func BenchmarkBNSD(b *testing.B) {
	var (
		alice = weavetest.NewKey()
		benny = weavetest.NewKey()
		carol = weavetest.NewKey()
		david = weavetest.NewKey()
	)

	type dict map[string]interface{}
	genesis := dict{
		"cash": []interface{}{
			dict{
				"address": alice.PublicKey().Condition().Address(),
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
		"distribution": []interface{}{
			dict{
				"admin": alice.PublicKey().Condition().Address(),
				"recipients": []interface{}{
					dict{"weight": 1, "address": benny.PublicKey().Condition().Address()},
				},
			},
		},
		"gconf": map[string]interface{}{
			cash.GconfCollectorAddress: hex.EncodeToString(david.PublicKey().Condition().Address()),
			cash.GconfMinimalFee:       coin.Coin{}, // no fee
		},
	}

	cases := map[string]struct {
		ops         func(weavetest.WeaveApp) error
		wantChanged bool
	}{
		"empty block": {
			ops: func(weavetest.WeaveApp) error {
				// Without sleep this test is locking the CPU.
				time.Sleep(time.Microsecond * 1000)
				return nil
			},
			wantChanged: false,
		},
		"send coins from alice to carol": {
			ops: func(wapp weavetest.WeaveApp) error {
				tx := Tx{
					Sum: &Tx_SendMsg{
						&cash.SendMsg{
							Src:    alice.PublicKey().Condition().Address(),
							Dest:   carol.PublicKey().Condition().Address(),
							Amount: coin.NewCoinp(0, 100, "IOV"),
						},
					},
				}

				nonce, err := getNonce(wapp, alice.PublicKey().Condition().Address())
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
				return nil
			},
			wantChanged: true,
		},
	}

	for testName, tc := range cases {
		b.Run(testName, func(b *testing.B) {
			bnsd := newBnsd(b)
			runner := weavetest.NewWeaveRunner(b, bnsd, "mychain")
			runner.InitChain(genesis)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				changed := runner.InBlock(tc.ops)
				if changed != tc.wantChanged {
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
	// t.Logf("using home directory: %q", homeDir)
	bnsd, err := GenerateApp(homeDir, log.NewNopLogger(), false)
	if err != nil {
		t.Fatalf("cannot generate bnsd instance: %s", err)
	}
	return bnsd
}

func getNonce(db weave.ReadOnlyKVStore, addr weave.Address) (int64, error) {
	obj, err := sigs.NewBucket().Get(db, addr)
	if err != nil {
		return 0, errors.Wrap(err, "cannot query nonce")
	}
	user := sigs.AsUser(obj)

	// Nonce not found
	if user == nil {
		return 0, nil
	}
	// Otherwise, read the nonce
	return user.Sequence, nil
}
