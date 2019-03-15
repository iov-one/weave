package app

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/iov-one/weave"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
)

func BenchmarkPerformance(b *testing.B) {
	// TODO genesis

	bnsd := newTestApplication(b)

	ah := appHelper{
		ChainID: "mychain",
		t:       b,
		app:     bnsd,
	}

	b.ResetTimer()
	fmt.Printf("Running with %d\n", b.N)
	for i := 0; i < b.N; i++ {
		changed := ah.InBlock(emptyBlock)
		if changed {
			b.Fatalf("Should not change on empty block")
		}
	}
}

func emptyBlock(app runner) error {
	time.Sleep(time.Microsecond * 100)
	return nil
}

func sendOneCoin(app runner) error {
	return app.DeliverTx(nil)
}

type runner interface {
	DeliverTx(tx weave.Tx) error
	CheckTx(tx weave.Tx) error
}

func newTestApplication(t tester) abci.Application {
	t.Helper()

	homeDir, err := ioutil.TempDir("", "bnsd_performance_home")
	if err != nil {
		t.Fatalf("cannot create a temporary directory: %s", err)
	}
	t.Logf("using home directory: %q", homeDir)
	bnsd, err := GenerateApp(homeDir, log.NewNopLogger(), false)
	if err != nil {
		t.Fatalf("cannot generate bnsd instance: %s", err)
	}
	return bnsd
}

type tester interface {
	Helper()
	Errorf(string, ...interface{})
	Fatalf(string, ...interface{})
	Logf(string, ...interface{})
}

type appHelper struct {
	ChainID string
	Height  int64
	t       tester
	app     abci.Application
}

var _ runner = (*appHelper)(nil)

func (b *appHelper) CheckTx(tx weave.Tx) error {
	bz, err := tx.Marshal()
	if err != nil {
		return err
	}
	res := b.app.CheckTx(bz)
	if res.Code != 0 {
		// take the result and turn it into an Error instance
		// or just start with "(res.code) res.log"
		return fmt.Errorf("%d: %s", res.Code, res.Log)
	}
	return nil
}

func (b *appHelper) DeliverTx(tx weave.Tx) error {
	panic("todo")
}

type processBlock func(r runner) error

// runs a block and returns a bool defining if the app hash (state)
// changed after commiting this processBlock
func (b *appHelper) InBlock(cb processBlock) bool {
	b.t.Helper()

	b.Height++
	blockHash := b.app.Info(abci.RequestInfo{}).LastBlockAppHash

	// BeginBlock will panic on error.
	b.app.BeginBlock(abci.RequestBeginBlock{
		Header: abci.Header{
			ChainID: b.ChainID,
			Height:  b.Height,
		},
	})

	if err := cb(b); err != nil {
		b.t.Fatalf("operation failed with %+v", err)
	}

	// EndBlock returns Validator diffs mainly,
	// but not important for benchmarks just tests
	b.app.EndBlock(abci.RequestEndBlock{
		Height: b.Height,
	})

	// Commit returns new app hash... maybe we can check if this hash
	// changed since last block and return that info here?
	finalHash := b.app.Commit().Data
	return !bytes.Equal(blockHash, finalHash)
}
