package app

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	abci "github.com/tendermint/tendermint/abci/types"
)

// BaseApp adds DeliverTx, CheckTx, and BeginBlock
// handlers to the storage and query functionality of StoreApp
type BaseApp struct {
	*StoreApp
	decoder weave.TxDecoder
	handler weave.Handler
	ticker  weave.Ticker
	debug   bool
}

var _ abci.Application = BaseApp{}

// NewBaseApp constructs a basic abci application
func NewBaseApp(
	store *StoreApp,
	decoder weave.TxDecoder,
	handler weave.Handler,
	ticker weave.Ticker,
	debug bool,
) BaseApp {
	return BaseApp{
		StoreApp: store,
		decoder:  decoder,
		handler:  handler,
		ticker:   ticker,
		debug:    debug,
	}
}

// DeliverTx - ABCI - dispatches to the handler
func (b BaseApp) DeliverTx(txBytes []byte) abci.ResponseDeliverTx {
	tx, err := b.loadTx(txBytes)
	if err != nil {
		return weave.DeliverTxError(err, b.debug)
	}

	// ignore error here, allow it to be logged
	ctx := weave.WithLogInfo(b.BlockContext(),
		"call", "deliver_tx",
		"path", weave.GetPath(tx))

	res, err := b.handler.Deliver(ctx, b.DeliverStore(), tx)
	if err == nil {
		b.AddValChange(res.Diff)
	}
	return weave.DeliverOrError(res, err, b.debug)
}

// CheckTx - ABCI - dispatches to the handler
func (b BaseApp) CheckTx(txBytes []byte) abci.ResponseCheckTx {
	tx, err := b.loadTx(txBytes)
	if err != nil {
		return weave.CheckTxError(err, b.debug)
	}

	ctx := weave.WithLogInfo(b.BlockContext(),
		"call", "check_tx",
		"path", weave.GetPath(tx))

	res, err := b.handler.Check(ctx, b.CheckStore(), tx)
	return weave.CheckOrError(res, err, b.debug)
}

// BeginBlock - ABCI
func (b BaseApp) BeginBlock(req abci.RequestBeginBlock) (res abci.ResponseBeginBlock) {
	// default: set the context properly
	b.StoreApp.BeginBlock(req)

	if b.ticker != nil {
		ctx := weave.WithLogInfo(b.BlockContext(), "call", "begin_block")
		res, err := b.ticker.Tick(ctx, b.DeliverStore())
		if err != nil {
			panic(err)
		}
		b.StoreApp.AddValChange(res.Diff)
	}
	return abci.ResponseBeginBlock{}
}

// loadTx calls the decoder, and capture any panics
func (b BaseApp) loadTx(txBytes []byte) (tx weave.Tx, err error) {
	defer errors.Recover(&err)
	tx, err = b.decoder(txBytes)
	return
}
