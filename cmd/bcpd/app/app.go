/*
Package app links together all the various components
to construct a bcp-demo app.
*/
package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store/iavl"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/escrow"
	"github.com/iov-one/weave/x/hashlock"
	"github.com/iov-one/weave/x/namecoin"
	"github.com/iov-one/weave/x/nft/username"
	"github.com/iov-one/weave/x/sigs"
	"github.com/iov-one/weave/x/utils"
)

// Authenticator returns the typical authentication,
// just using public key signatures
func Authenticator() x.Authenticator {
	return x.ChainAuth(sigs.Authenticate{}, hashlock.Authenticate{})
}

// Chain returns a chain of decorators, to handle authentication,
// fees, logging, and recovery
func Chain(minFee x.Coin, authFn x.Authenticator) app.Decorators {
	return app.ChainDecorators(
		utils.NewLogging(),
		utils.NewRecovery(),
		utils.NewKeyTagger(),
		// on CheckTx, bad tx don't affect state
		utils.NewSavepoint().OnCheck(),
		sigs.NewDecorator(),
		namecoin.NewFeeDecorator(authFn, minFee),
		// cannot pay for fee with hashlock...
		hashlock.NewDecorator(),
		// on DeliverTx, bad tx will increment nonce and take fee
		// even if the message fails
		utils.NewSavepoint().OnDeliver(),
	)
}

// Router returns a default router, only dispatching to the
// cash.SendMsg
func Router(authFn x.Authenticator, issuer weave.Address) app.Router {
	r := app.NewRouter()
	namecoin.RegisterRoutes(r, authFn, issuer)
	// we use the namecoin wallet handler
	// TODO: move to cash upon refactor
	escrow.RegisterRoutes(r, authFn, namecoin.NewController())
	username.RegisterRoutes(r, authFn, issuer)
	return r
}

// QueryRouter returns a default query router,
// allowing access to "/wallets", "/auth", "/", and "/escrows"
func QueryRouter() weave.QueryRouter {
	r := weave.NewQueryRouter()
	r.RegisterAll(
		escrow.RegisterQuery,
		namecoin.RegisterQuery,
		sigs.RegisterQuery,
		username.RegisterQuery,
		orm.RegisterQuery,
	)
	return r
}

// Stack wires up a standard router with a standard decorator
// chain. This can be passed into BaseApp.
func Stack(minFee x.Coin, issuer weave.Address) weave.Handler {
	authFn := Authenticator()
	return Chain(minFee, authFn).
		WithHandler(Router(authFn, issuer))
}

// Application constructs a basic ABCI application with
// the given arguments. If you are not sure what to use
// for the Handler, just use Stack().
func Application(name string, h weave.Handler,
	tx weave.TxDecoder, dbPath string, debug bool) (app.BaseApp, error) {

	ctx := context.Background()
	kv, err := CommitKVStore(dbPath)
	if err != nil {
		return app.BaseApp{}, err
	}
	store := app.NewStoreApp(name, kv, QueryRouter(), ctx)
	base := app.NewBaseApp(store, tx, h, nil, debug)
	return base, nil
}

// CommitKVStore returns an initialized KVStore that persists
// the data to the named path.
func CommitKVStore(dbPath string) (weave.CommitKVStore, error) {
	// memory backed case, just for testing
	if dbPath == "" {
		return iavl.MockCommitStore(), nil
	}

	// Expand the path fully
	path, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, fmt.Errorf("Invalid Database Name: %s", path)
	}

	// Some external calls accidently add a ".db", which is now removed
	path = strings.TrimSuffix(path, filepath.Ext(path))

	// Split the database name into it's components (dir, name)
	dir := filepath.Dir(path)
	name := filepath.Base(path)
	return iavl.NewCommitStore(dir, name), nil
}
