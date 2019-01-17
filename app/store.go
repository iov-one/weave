package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
)

// StoreApp contains a data store and all info needed
// to perform queries and handshakes.
//
// It should be embedded in another struct for CheckTx,
// DeliverTx and initializing state from the genesis.
// Errors on ABCI steps handled as panics
// I'm sorry Alex, but there is no other way :(
// https://github.com/tendermint/tendermint/abci/issues/165#issuecomment-353704015
// "Regarding errors in general, for messages that don't take
//  user input like Flush, Info, InitChain, BeginBlock, EndBlock,
// and Commit.... There is no way to handle these errors gracefully,
// so we might as well panic."
type StoreApp struct {
	logger log.Logger

	// name is what is returned from abci.Info
	name string

	// Database state (committed, check, deliver....)
	store *CommitStore

	// Code to initialize from a genesis file
	initializer weave.Initializer

	// How to handle queries
	queryRouter weave.QueryRouter

	// chainID is loaded from db in initialization
	// saved once in parseGenesis
	chainID string

	// cached validator changes from DeliverTx
	pending []abci.ValidatorUpdate

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
func NewStoreApp(name string, store weave.CommitKVStore,
	queryRouter weave.QueryRouter, baseContext weave.Context) *StoreApp {
	s := &StoreApp{
		name: name,
		// note: panics if trouble initializing from store
		store:       NewCommitStore(store),
		queryRouter: queryRouter,
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

// WithInit is used to set the init function we call
func (s *StoreApp) WithInit(init weave.Initializer) *StoreApp {
	s.initializer = init
	return s
}

// parseAppState is called from InitChain, the first time the chain
// starts, and not on restarts.
func (s *StoreApp) parseAppState(data []byte, chainID string, init weave.Initializer) error {
	if s.chainID != "" {
		return fmt.Errorf("appState previously loaded for chain: %s", s.chainID)
	}

	if len(data) == 0 {
		return fmt.Errorf("app_state not set in genesis.json, please initialize application before launching the blockchain")
	}

	var appState weave.Options
	err := json.Unmarshal(data, &appState)
	if err != nil {
		return errors.WithCode(err, errors.CodeTxParseError)
	}

	err = s.storeChainID(chainID)
	if err != nil {
		return err
	}

	return init.FromGenesis(appState, s.DeliverStore())
}

// store chainID and update context
func (s *StoreApp) storeChainID(chainId string) error {
	// set the chainID
	s.chainID = chainId
	err := saveChainID(s.DeliverStore(), s.chainID)
	if err != nil {
		return err
	}
	// and update the context
	s.baseContext = weave.WithChainID(s.baseContext, s.chainID)

	return nil
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

/*
Query gets data from the app store.
A query request has the following elements:
* Path - the type of query
* Data - what to query, interpretted based on Path
* Height - the block height to query (if 0 most recent)
* Prove - if true, also return a proof

Path may be "/", "/<bucket>", or "/<bucket>/<index>"
It may be followed by "?prefix" to make a prefix query.
Soon we will support "?range" for powerful range queries

Key and Value in Results are always serialized ResultSet
objects, able to support 0 to N values. They must be the
same size. This makes things a little more difficult for
simple queries, but provides a consistent interface.
*/
func (s *StoreApp) Query(reqQuery abci.RequestQuery) (resQuery abci.ResponseQuery) {

	// find the handler
	path, mod := splitPath(reqQuery.Path)
	qh := s.queryRouter.Handler(path)
	if qh == nil {
		resQuery.Code = errors.CodeUnknownRequest
		resQuery.Log = fmt.Sprintf("Unexpected Query path: %v", reqQuery.Path)
		return
	}

	// TODO: support historical queries by getting old read-only
	// height := reqQuery.Height
	// if height == 0 {
	// 	withProof := s.CommittedHeight() - 1
	// 	if tree.Tree.VersionExists(uint64(withProof)) {
	// 		height = withProof
	// 	} else {
	// 		height = s.CommittedHeight()
	// 	}
	// }
	height, _ := s.store.CommitInfo()
	resQuery.Height = height
	// TODO: better version handling!
	db := s.store.committed.CacheWrap()

	// make the query
	models, err := qh.Query(db, mod, reqQuery.Data)
	if err != nil {
		return queryError(err)
	}

	// set the info as ResultSets....
	resQuery.Key, err = ResultsFromKeys(models).Marshal()
	if err != nil {
		return queryError(err)
	}
	resQuery.Value, err = ResultsFromValues(models).Marshal()
	if err != nil {
		return queryError(err)
	}

	// TODO: support proofs given this info....
	// if reqQuery.Prove {
	//  value, proof, err := tree.GetVersionedWithProof(key, height)
	//  if err != nil {
	//      resQuery.Log = err.Error()
	//      break
	//  }
	//  resQuery.Value = value
	//  resQuery.Proof = proof.Bytes()

	return resQuery
}

// splitPath splits out the real path along with the query
// modifier (everything after the ?)
func splitPath(path string) (string, string) {
	var mod string
	chunks := strings.SplitN(path, "?", 2)
	if len(chunks) == 2 {
		path = chunks[0]
		mod = chunks[1]
	}
	return path, mod
}

func queryError(err error) abci.ResponseQuery {
	return abci.ResponseQuery{
		Log:  err.Error(),
		Code: errors.CodeInternalErr,
	}
}

// Commit implements abci.Application
func (s *StoreApp) Commit() (res abci.ResponseCommit) {
	commitID := s.store.Commit()

	s.logger.Debug("Commit synced",
		"height", commitID.Version,
		"hash", fmt.Sprintf("%X", commitID.Hash),
	)

	return abci.ResponseCommit{Data: commitID.Hash}
}

// InitChain implements ABCI
// TODO: store the original validators somewhere
// Note: in tendermint 0.17, the genesis file is passed
// in here, we should use this to trigger reading the genesis now
// TODO: investigate validators and consensusParams in response
func (s *StoreApp) InitChain(req abci.RequestInitChain) (res abci.ResponseInitChain) {
	err := s.parseAppState(req.AppStateBytes, req.ChainId, s.initializer)
	if err != nil {
		// Read comment on type header
		panic(err)
	}

	return abci.ResponseInitChain{}
}

// BeginBlock implements ABCI
// Sets up blockContext
// TODO: investigate response tags as of 0.11 abci
func (s *StoreApp) BeginBlock(req abci.RequestBeginBlock) (res abci.ResponseBeginBlock) {
	// set the begin block context
	ctx := weave.WithHeader(s.baseContext, req.Header)
	ctx = weave.WithHeight(ctx, req.Header.GetHeight())
	s.blockContext = ctx

	return
}

// EndBlock - ABCI
// Returns a list of all validator changes made in this block
// TODO: investigate response tags as of 0.11 abci
func (s *StoreApp) EndBlock(_ abci.RequestEndBlock) (res abci.ResponseEndBlock) {
	res.ValidatorUpdates = s.pending
	s.pending = nil
	return
}

// AddValChange is meant to be called by apps on DeliverTx
// results, this is added to the cache for the endblock changeset
func (s *StoreApp) AddValChange(diffs []abci.ValidatorUpdate) {
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

// return index of list with validator of same Pubkey, or -1 if no match
func pubKeyIndex(val abci.ValidatorUpdate, list []abci.ValidatorUpdate) int {
	for i, v := range list {
		if val.PubKey.Type == v.PubKey.Type && bytes.Equal(val.PubKey.Data, v.PubKey.Data) {
			return i
		}
	}
	return -1
}
