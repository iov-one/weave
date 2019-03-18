package weavetest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	abci "github.com/tendermint/tendermint/abci/types"
)

// Tester is implemented by both *testing.T and *testing.B. Use it instead of
// the pointer type to allow notation to accept both objects.
type Tester interface {
	Helper()
	Errorf(string, ...interface{})
	Fatalf(string, ...interface{})
	Logf(string, ...interface{})
}

// WeaveRunner provides a translation layer between an ABCI interface and a
// weave application. It takes care of serializing messages and creating
// blocks.
type WeaveRunner struct {
	chainID string
	height  int64
	t       Tester
	app     abci.Application
}

// NewWeaveRunner creates a WeaveRunner instance that can be used to process
// deliver and check transaction requests using weave API. This runner expects
// all operations to succeed. Any error results in test failure.
func NewWeaveRunner(t Tester, app abci.Application, chainID string) *WeaveRunner {
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

	// Load the genesis in a separate block.
	changed := w.InBlock(func(WeaveApp) error {
		w.app.InitChain(abci.RequestInitChain{
			Time:          time.Now(),
			ChainId:       w.chainID,
			AppStateBytes: raw,
		})
		return nil
	})

	if !changed {
		w.t.Fatalf("genesis did not change the state")
	}
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
	// TODO: avoid importing app
	var value app.ResultSet
	err := value.Unmarshal(query.Value)
	if err != nil {
		// oh, for an error return here...
		panic(errors.Wrap(err, "cannot parse values"))
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
	// if only the interface supported returning errors....
	if query.Code != 0 {
		panic(query.Log)
	}
	models, err := toModels(query.Key, query.Value)
	if err != nil {
		// oh, for an error return here...
		panic(errors.Wrap(err, "cannot parse values"))
	}

	// TODO: remove store dependency
	return store.NewSliceIterator(models)
}

func (w *WeaveRunner) ReverseIterator(start, end []byte) weave.Iterator {
	// TODO: load normal iterator but then play it backwards?
	panic("not implemented")
}

// TODO: we really don't want to import weave/app here, do we... but we need it to parse
func toModels(keys []byte, values []byte) ([]weave.Model, error) {
	var k, v app.ResultSet
	err := k.Unmarshal(keys)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse keys")
	}
	err = v.Unmarshal(values)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse values")
	}
	return app.JoinResults(&k, &v)
}
