package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/tmlibs/log"

	"github.com/confio/weave"
	"github.com/confio/weave/errors"
)

// StoreApp contains a data store and all info needed
// to perform queries and handshakes.
//
// It should be embedded in another struct for CheckTx,
// DeliverTx and initializing state from the genesis.
type StoreApp struct {
	logger log.Logger

	// name is what is returned from abci.Info
	name string

	// Database state (committed, check, deliver....)
	store *commitStore

	// Code to initialize from a genesis file
	initializer weave.Initializer

	// chainID is loaded from db in initialization
	// saved once in parseGenesis
	chainID string

	// cached validator changes from DeliverTx
	pending []*abci.Validator

	// baseContext contains context info that is valid for
	// lifetime of this app (eg. chainID)
	baseContext weave.Context

	// blockContext contains context info that is valid for the
	// current block (eg. height, header), reset on BeginBlock
	blockContext weave.Context

	// genesisFile (temporary) is used to store the file
	// to read from on InitChain
	genesisFile string
}

// NewStoreApp initializes this app into a ready state with some defaults
//
// panics if unable to properly load the state from the given store
// TODO: is this correct? nothing else to do really....
func NewStoreApp(name string, store weave.CommitKVStore, baseContext weave.Context) *StoreApp {
	s := &StoreApp{
		name: name,
		// note: panics if trouble initializing from store
		store:       newCommitStore(store),
		baseContext: baseContext,
	}
	s = s.WithLogger(log.NewNopLogger())

	// load the chainID from the db
	s.chainID = loadChainID(s.DeliverStore())
	if s.chainID != "" {
		s.baseContext = weave.WithChainID(s.baseContext, s.chainID)
	}

	// get the most recent height
	height, _ := s.store.CommitInfo()
	s.blockContext = weave.WithHeight(s.baseContext, height)
	return s
}

// GetChainID returns the current chainID
func (s *StoreApp) GetChainID() string {
	return s.chainID
}

// WithGenesis is used to set the genesis file we read
// initial state from, until it is sent from tendermint
func (s *StoreApp) WithGenesis(genesisFile string) *StoreApp {
	s.genesisFile = genesisFile
	return s
}

// WithInit is used to set the init function we call
func (s *StoreApp) WithInit(init weave.Initializer) *StoreApp {
	s.initializer = init
	return s
}

// parseGenesis is called from InitChain, the first time the chain
// starts, and not on restarts.
//
// It currently take bytes from local genesis file, soon, it will
// receive genesis bytes from tendermint
func (s *StoreApp) parseGenesis(data []byte, init weave.Initializer) error {
	if s.chainID != "" {
		return fmt.Errorf("Genesis file previously loaded for chain: %s", s.chainID)
	}

	var gen Genesis
	err := json.Unmarshal(data, &gen)
	if err != nil {
		return errors.WithCode(err, errors.CodeTxParseError)
	}

	// set the chainID from the genesis file
	s.chainID = gen.ChainID
	err = saveChainID(s.DeliverStore(), s.chainID)
	if err != nil {
		return err
	}
	// and update the context
	s.baseContext = weave.WithChainID(s.baseContext, s.chainID)

	return init.FromGenesis(gen.AppState, s.DeliverStore())
}

// WithLogger sets the logger on the StoreApp and returns it,
// to make it easy to chain in initialization
//
// also sets baseContext logger
func (s *StoreApp) WithLogger(logger log.Logger) *StoreApp {
	s.baseContext = weave.WithLogger(s.baseContext, logger)
	s.logger = logger
	return s
}

// Logger returns the application base logger
func (s *StoreApp) Logger() log.Logger {
	return s.logger
}

// BlockContext returns the block context for public use
func (s *StoreApp) BlockContext() weave.Context {
	return s.blockContext
}

// DeliverStore returns the current DeliverTx cache for methods
func (s *StoreApp) DeliverStore() weave.CacheableKVStore {
	return s.store.deliver
}

// CheckStore returns the current CheckTx cache for methods
func (s *StoreApp) CheckStore() weave.CacheableKVStore {
	return s.store.check
}

//----------------------- ABCI ---------------------

// Info implements abci.Application. It returns the height and hash,
// as well as the abci name and version.
//
// The height is the block that holds the transactions, not the apphash itself.
func (s *StoreApp) Info(req abci.RequestInfo) abci.ResponseInfo {
	height, hash := s.store.CommitInfo()

	s.logger.Info("Info synced",
		"height", height,
		"hash", fmt.Sprintf("%X", hash))

	return abci.ResponseInfo{
		Data:             s.name,
		LastBlockHeight:  height,
		LastBlockAppHash: hash,
	}
}

// SetOption - ABCI
// TODO: not implemented (ABCI spec still unclear....)
func (s *StoreApp) SetOption(res abci.RequestSetOption) abci.ResponseSetOption {
	return abci.ResponseSetOption{Log: "Not Implemented"}
}

// Query - ABCI
func (s *StoreApp) Query(reqQuery abci.RequestQuery) (resQuery abci.ResponseQuery) {
	if len(reqQuery.Data) == 0 {
		resQuery.Log = "Query cannot be zero length"
		resQuery.Code = errors.CodeUnknownRequest
		return
	}

	// TODO: support historical queries

	// height := reqQuery.Height
	// if height == 0 {
	// 	// TODO: once the rpc actually passes in non-zero
	// 	// heights we can use to query right after a tx
	// 	// we must retrun most recent, even if apphash
	// 	// is not yet in the blockchain

	// 	withProof := s.CommittedHeight() - 1
	// 	if tree.Tree.VersionExists(uint64(withProof)) {
	// 		height = withProof
	// 	} else {
	// 		height = s.CommittedHeight()
	// 	}
	// }
	height, _ := s.store.CommitInfo()
	resQuery.Height = height

	switch reqQuery.Path {
	case "/store", "/key": // Get by key
		key := reqQuery.Data // Data holds the key bytes
		resQuery.Key = key
		// TODO: support proofs

		// if reqQuery.Prove {
		// 	value, proof, err := tree.GetVersionedWithProof(key, height)
		// 	if err != nil {
		// 		resQuery.Log = err.Error()
		// 		break
		// 	}
		// 	resQuery.Value = value
		// 	resQuery.Proof = proof.Bytes()
		// } else {
		value := s.store.committed.Get(key)
		resQuery.Value = value
	default:
		resQuery.Code = errors.CodeUnknownRequest
		resQuery.Log = fmt.Sprintf("Unexpected Query path: %v", reqQuery.Path)
	}
	return
}

// Commit implements abci.Application
func (s *StoreApp) Commit() (res abci.ResponseCommit) {
	commitID := s.store.Commit()

	s.logger.Debug("Commit synced",
		"height", commitID.Version,
		"hash", fmt.Sprintf("%X", commitID.Hash),
	)

	// TODO: needed???
	// if s.state.Size() == 0 {
	// 	return abci.ResponseCommit{Log: "Empty hash for empty tree"}
	// }
	return abci.ResponseCommit{Data: commitID.Hash}
}

// InitChain implements ABCI
// TODO: store the original validators somewhere
// Note: in tendermint 0.17, the genesis file is passed
// in here, we should use this to trigger reading the genesis now
func (s *StoreApp) InitChain(req abci.RequestInitChain) (res abci.ResponseInitChain) {
	bz, err := ioutil.ReadFile(s.genesisFile)
	if err != nil {
		// I'm sorry Alex, but there is no other way :(
		// https://github.com/tendermint/abci/issues/165#issuecomment-353704015
		// "Regarding errors in general, for messages that don't take
		//  user input like Flush, Info, InitChain, BeginBlock, EndBlock,
		// and Commit.... There is no way to handle these errors gracefully,
		// so we might as well panic."
		panic(err)
	}

	return s.InitChainWithGenesis(req, bz)
}

// InitChainWithGenesis mocks out what we want for tendermint 0.17
func (s *StoreApp) InitChainWithGenesis(req abci.RequestInitChain,
	data []byte) abci.ResponseInitChain {

	err := s.parseGenesis(data, s.initializer)
	if err != nil {
		// I know, read above...
		panic(err)
	}
	return abci.ResponseInitChain{}
}

// BeginBlock implements ABCI
// Sets up blockContext
func (s *StoreApp) BeginBlock(req abci.RequestBeginBlock) (res abci.ResponseBeginBlock) {
	// set the begin block context
	ctx := weave.WithHeader(s.baseContext, *req.Header)
	ctx = weave.WithHeight(ctx, req.Header.GetHeight())
	s.blockContext = ctx

	return
}

// EndBlock - ABCI
// Returns a list of all validator changes made in this block
func (s *StoreApp) EndBlock(_ abci.RequestEndBlock) (res abci.ResponseEndBlock) {
	res.ValidatorUpdates = s.pending
	s.pending = nil
	return
}

// AddValChange is meant to be called by apps on DeliverTx
// results, this is added to the cache for the endblock changeset
func (s *StoreApp) AddValChange(diffs []*abci.Validator) {
	// ensures multiple updates for one validator are combined into one slot
	for _, d := range diffs {
		idx := pubKeyIndex(d, s.pending)
		if idx >= 0 {
			s.pending[idx] = d
		} else {
			s.pending = append(s.pending, d)
		}
	}
}

// return index of list with validator of same PubKey, or -1 if no match
func pubKeyIndex(val *abci.Validator, list []*abci.Validator) int {
	for i, v := range list {
		if bytes.Equal(val.PubKey, v.PubKey) {
			return i
		}
	}
	return -1
}
