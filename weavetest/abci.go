package weavetest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	abci "github.com/tendermint/tendermint/abci/types"
)

// Strategy defines which functions we call in ProcessAllTxs.
type Strategy uint8

// Has return true if this strategy contains given one - given strategy is a
// subset of this one.
func (s Strategy) Has(other Strategy) bool {
	return s&other != 0
}

const (
	// When set the CheckTx method is being executed for each transaction.
	ExecCheck Strategy = 1 << iota

	// When set and running as part of the benchmark, if the CheckTx method
	// is executed then it is excluded from measurements. Benchmarks are
	// being paused for the execution of the CheckTx.
	NoBenchCheck

	// When set the DeliverTx method is being executed for each
	// transaction.
	ExecDeliver

	ExecCheckAndDeliver = ExecCheck | ExecDeliver
)

// WeaveRunner provides a translation layer between an ABCI interface and a
// weave application. It takes care of serializing messages and creating
// blocks.
type WeaveRunner struct {
	chainID string
	height  int64
	t       testing.TB
	app     abci.Application
}

// NewWeaveRunner creates a WeaveRunner instance that can be used to process
// deliver and check transaction requests using weave API. This runner expects
// all operations to succeed. Any error results in test failure.
func NewWeaveRunner(t testing.TB, app abci.Application, chainID string) *WeaveRunner {
	return &WeaveRunner{
		chainID: chainID,
		height:  0,
		t:       t,
		app:     app,
	}
}

// WeaveApp is implemented by a weave application. This is the minimal
// interface required by the WeaveRunner to be able to connect ABCI and weave
// APIs together.
type WeaveApp interface {
	DeliverTx(weave.Tx) error
	CheckTx(weave.Tx) error
}

// InitChain serialize to JSON given genesis and loads it. Loading a genesis is
// causing a block creation.
func (w *WeaveRunner) InitChain(genesis interface{}) {
	raw, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		w.t.Fatalf("cannot JSON serialize genesis: %s", err)
	}

	// InitChain must happen before any blocks are created
	lastHeight := w.app.Info(abci.RequestInfo{}).LastBlockHeight
	if lastHeight != 0 {
		w.t.Fatalf("cannot initialize after a block, height=%d", lastHeight)
	}
	w.app.InitChain(abci.RequestInitChain{
		Time:          time.Now(),
		ChainId:       w.chainID,
		AppStateBytes: raw,
	})

	// create initial block to commit state
	w.InBlock(func(_ WeaveApp) error { return nil })
}

// CheckTx translates given weave transaction into ABCI interface and executes.
func (w *WeaveRunner) CheckTx(tx weave.Tx) error {
	raw, err := tx.Marshal()
	if err != nil {
		return errors.Wrap(err, "cannot marshal transaction")
	}
	if resp := w.app.CheckTx(raw); resp.Code != 0 {
		return fmt.Errorf("%d: %s", resp.Code, resp.Log)
	}
	return nil
}

// DeliverTx translates given weave transaction into ABCI interface and
// executes.
func (w *WeaveRunner) DeliverTx(tx weave.Tx) error {
	raw, err := tx.Marshal()
	if err != nil {
		return errors.Wrap(err, "cannot marshal transaction")
	}
	if resp := w.app.DeliverTx(raw); resp.Code != 0 {
		return fmt.Errorf("%d: %s", resp.Code, resp.Log)
	}
	return nil
}

// InBlock begins a block and runs given function. All transactions executed
// withing given function are part of newly created block. Upon success the
// block is finished and changes committed.
// InBlock returns true if the application state was modified. It returns false
// if creating new block did not modify the state.
//
// Any failure is ending the test instantly.
func (w *WeaveRunner) InBlock(executeTx func(WeaveApp) error) bool {
	w.t.Helper()

	w.height++

	initialHash := w.app.Info(abci.RequestInfo{}).LastBlockAppHash

	// BeginBlock will panic on error.
	w.app.BeginBlock(abci.RequestBeginBlock{
		Header: abci.Header{
			ChainID: w.chainID,
			Height:  w.height,
		},
	})

	if err := executeTx(w); err != nil {
		w.t.Fatalf("operation failed with %+v", err)
	}

	// EndBlock returns Validator diffs mainly,
	// but not important for benchmarks just tests
	w.app.EndBlock(abci.RequestEndBlock{
		Height: w.height,
	})

	// Commit data contains the new app hash. It differs from the initial
	// hash only if the state was modified.
	finalHash := w.app.Commit().Data
	return !bytes.Equal(initialHash, finalHash)
}

// ProcessAllTxs will run all included txs, split into blocksize.
// It will Fail() if any tx returns an error, or if at any block,
// the appHash does not change (if should change, otherwise, require it stable)
func (w *WeaveRunner) ProcessAllTxs(blocks [][]weave.Tx, st Strategy) {
	// When running as part of a benchmark an additional functionality can
	// be provided.
	b, isBench := w.t.(*testing.B)

	for _, txs := range blocks {
		changed := w.InBlock(func(wapp WeaveApp) error {
			if st.Has(ExecCheck) {
				if isBench && st.Has(NoBenchCheck) {
					b.StopTimer()
				}
				for _, tx := range txs {
					if err := wapp.CheckTx(tx); err != nil {
						return errors.Wrap(err, "cannot check tx")
					}
				}
				if isBench && st.Has(NoBenchCheck) {
					b.StartTimer()
				}
			}

			if st.Has(ExecDeliver) {
				for _, tx := range txs {
					if err := wapp.DeliverTx(tx); err != nil {
						return errors.Wrap(err, "cannot deliver tx")
					}
				}
			}
			return nil

		})

		// If a delivery was made then the state must have changed.
		wantChanged := st.Has(ExecDeliver)
		if changed != wantChanged {
			w.t.Fatalf("expected state to change: %v", wantChanged)
		}
	}
}

// SplitTxs will break one slice of transactions into many slices,
// one per block. It will fill up to txPerBlock txs in each block
// The last block may have less, if there is not enough for a full block
func SplitTxs(txs []weave.Tx, txPerBlock int) [][]weave.Tx {
	numBlocks := numBlocks(len(txs), txPerBlock)
	res := make([][]weave.Tx, numBlocks)

	// full chunks for all but the last block
	for i := 0; i < numBlocks-1; i++ {
		res[i], txs = txs[:txPerBlock], txs[txPerBlock:]
	}

	// remainder in the last block
	res[numBlocks-1] = txs
	return res
}

// numBlocks returns total number of blocks for benchmarks that split b.N
// into many smaller blocks
func numBlocks(totalTx, txPerBlock int) int {
	runs := totalTx / txPerBlock
	if totalTx%txPerBlock > 0 {
		return runs + 1
	}
	return runs
}
