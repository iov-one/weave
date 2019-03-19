package weavetest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	abci "github.com/tendermint/tendermint/abci/types"
)

// Strategy defines which functions we call in ProcessAllTxs
type Strategy int

const (
	// CheckOnly will only run Check
	CheckOnly Strategy = iota
	// DeliverOnly will only run Deliver
	DeliverOnly
	// CheckAndDeliver will call Check for all txs in a block, then Deliver for all txs
	CheckAndDeliver
	// DeliverWithPrecheck will call Check *with timer stopped* then time Deliver for all txs
	DeliverWithPrecheck
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
	// we also allow standard queries... wrap into a bucket for ease of use
	weave.ReadOnlyKVStore
}

var _ WeaveApp = (*WeaveRunner)(nil)

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
// block is finished and changes commited.
// InBlock returns true if the application state was modified. It returns true
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

func applyStategy(t testing.TB, txs []weave.Tx, strategy Strategy) func(WeaveApp) error {
	b, ok := t.(*testing.B)
	freezeCheck := ok && strategy == DeliverWithPrecheck

	return func(wapp WeaveApp) error {
		// let's do all CheckTx first
		if strategy != DeliverOnly {
			if freezeCheck {
				b.StopTimer()
			}
			for _, tx := range txs {
				if err := wapp.CheckTx(tx); err != nil {
					return errors.Wrap(err, "cannot check tx")
				}
			}
			if freezeCheck {
				b.StartTimer()
			}
		}
		// then all DeliverTx... as would be done in reality
		if strategy != CheckOnly {
			for _, tx := range txs {
				if err := wapp.DeliverTx(tx); err != nil {
					return errors.Wrap(err, "cannot deliver tx")
				}
			}
		}
		return nil

	}
}

// ProcessAllTxs will run all included txs, split into blocksize.
// It will Fail() if any tx returns an error, or if at any block,
// the appHash does not change (if should change, otherwise, require it stable)
func (w *WeaveRunner) ProcessAllTxs(blocks [][]weave.Tx, strategy Strategy) {
	for _, txs := range blocks {
		changed := w.InBlock(applyStategy(w.t, txs, strategy))
		// no change on CheckOnly, otherwise there must be change
		if changed != (strategy != CheckOnly) {
			w.t.Fatal("expected state to change")
		}
	}
}

// SplitTxs will break one slice of transactions into many slices,
// one per block. It will fill up to txPerBlockx txs in each block
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

var _ weave.ReadOnlyKVStore = (*WeaveRunner)(nil)

func (w *WeaveRunner) Get(key []byte) []byte {
	query := w.app.Query(abci.RequestQuery{
		Path: "/",
		Data: key,
	})
	// if only the interface supported returning errors....
	if query.Code != 0 {
		panic(query.Log)
	}
	var value ResultSet
	if err := value.Unmarshal(query.Value); err != nil {
		panic(errors.Wrap(err, "unmarshal result set"))
	}
	if len(value.Results) == 0 {
		return nil
	}
	// TODO: assert error if len > 1 ???
	return value.Results[0]
}

func (w *WeaveRunner) Has(key []byte) bool {
	return len(w.Get(key)) > 0
}

func (w *WeaveRunner) Iterator(start, end []byte) weave.Iterator {
	// TODO: support all prefix searches (later even more ranges)
	// look at orm/query.go:prefixRange for an idea how we turn prefix->iterator,
	// we should detect this case and reverse it so we can serialize over abci query
	if start != nil || end != nil {
		panic("iterator only implemented for entire range")
	}

	query := w.app.Query(abci.RequestQuery{
		Path: "/?prefix",
		Data: nil,
	})
	if query.Code != 0 {
		panic(query.Log)
	}
	models, err := toModels(query.Key, query.Value)
	if err != nil {
		panic(errors.Wrap(err, "cannot convert to model"))
	}

	// TODO: remove store dependency
	return store.NewSliceIterator(models)
}

func (w *WeaveRunner) ReverseIterator(start, end []byte) weave.Iterator {
	// TODO: load normal iterator but then play it backwards?
	panic("not implemented")
}

func toModels(keys, values []byte) ([]weave.Model, error) {
	var k, v ResultSet
	if err := k.Unmarshal(keys); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal keys")
	}
	if err := v.Unmarshal(values); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal values")
	}

	if len(k.Results) != len(v.Results) {
		return nil, errors.Wrap(errors.ErrInvalidInput, "mismatches result set size")
	}

	mods := make([]weave.Model, len(k.Results))
	for i := range mods {
		mods[i] = weave.Model{
			Key:   k.Results[i],
			Value: v.Results[i],
		}
	}
	return mods, nil
}
