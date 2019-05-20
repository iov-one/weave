/*
Package app links together all the various components
to construct the bnsd app.
*/
package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/cmd/bnsd/x/nft/username"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/commands/server"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store/iavl"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/aswap"
	"github.com/iov-one/weave/x/batch"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/currency"
	"github.com/iov-one/weave/x/distribution"
	"github.com/iov-one/weave/x/escrow"
	gov "github.com/iov-one/weave/x/gov"
	"github.com/iov-one/weave/x/hashlock"
	"github.com/iov-one/weave/x/msgfee"
	"github.com/iov-one/weave/x/multisig"
	"github.com/iov-one/weave/x/nft"
	"github.com/iov-one/weave/x/nft/base"
	"github.com/iov-one/weave/x/sigs"
	"github.com/iov-one/weave/x/utils"
	"github.com/iov-one/weave/x/validators"
)

// Authenticator returns the typical authentication,
// just using public key signatures
func Authenticator() x.Authenticator {
	return x.ChainAuth(sigs.Authenticate{}, hashlock.Authenticate{}, multisig.Authenticate{})
}

// Chain returns a chain of decorators, to handle authentication,
// fees, logging, and recovery
func Chain(authFn x.Authenticator, minFee coin.Coin) app.Decorators {
	// ctrl can be initialized with any implementation, but must be used
	// consistently everywhere.
	var ctrl cash.Controller = cash.NewController(cash.NewBucket())

	return app.ChainDecorators(
		utils.NewLogging(),
		utils.NewRecovery(),
		utils.NewKeyTagger(),
		// on CheckTx, bad tx don't affect state
		utils.NewSavepoint().OnCheck(),
		sigs.NewDecorator(),
		multisig.NewDecorator(authFn),
		// cash.NewDynamicFeeDecorator embeds utils.NewSavepoint().OnDeliver()
		cash.NewDynamicFeeDecorator(authFn, ctrl),
		msgfee.NewAntispamFeeDecorator(minFee),
		msgfee.NewFeeDecorator(),
		// cannot pay for fee with hashlock...
		hashlock.NewDecorator(),
		batch.NewDecorator(),
	)
}

// Router returns a default router, only dispatching to the
// cash.SendMsg
func Router(authFn x.Authenticator, issuer weave.Address, nftBuckets map[string]orm.Bucket) app.Router {
	r := app.NewRouter()

	// ctrl can be initialized with any implementation, but must be used
	// consistently everywhere.
	var ctrl cash.Controller = cash.NewController(cash.NewBucket())

	migration.RegisterRoutes(r, authFn)
	cash.RegisterRoutes(r, authFn, ctrl)
	escrow.RegisterRoutes(r, authFn, ctrl)
	multisig.RegisterRoutes(r, authFn)
	//TODO: Possibly revisit passing the bucket later to have more control over types?
	// or implement a check
	currency.RegisterRoutes(r, authFn, issuer)
	username.RegisterRoutes(r, authFn, issuer)
	validators.RegisterRoutes(r, authFn)
	distribution.RegisterRoutes(r, authFn, ctrl)
	base.RegisterRoutes(r, authFn, issuer, nftBuckets)
	sigs.RegisterRoutes(r, authFn)
	aswap.RegisterRoutes(r, authFn, ctrl)
	gov.RegisterRoutes(r, authFn, decodeProposalOptions, proposalOptionsExecutor(ctrl))
	return r
}

// QueryRouter returns a default query router,
// allowing access to "/wallets", "/auth", "/", "/escrows", "/nft/usernames",
// "/nft/blockchains", "/nft/tickers", "/validators"
func QueryRouter(minFee coin.Coin) weave.QueryRouter {
	r := weave.NewQueryRouter()
	antiSpamQuery := msgfee.NewAntiSpamQuery(minFee)

	r.RegisterAll(
		migration.RegisterQuery,
		escrow.RegisterQuery,
		cash.RegisterQuery,
		sigs.RegisterQuery,
		multisig.RegisterQuery,
		username.RegisterQuery,
		validators.RegisterQuery,
		orm.RegisterQuery,
		currency.RegisterQuery,
		distribution.RegisterQuery,
		antiSpamQuery.RegisterQuery,
		aswap.RegisterQuery,
		gov.RegisterQuery,
	)
	return r
}

// Register nft types and actions for shared action handling via base handler
func RegisterNft() {
	// Default nft actions.
	nft.RegisterAction(nft.DefaultActions...)
}

// Stack wires up a standard router with a standard decorator
// chain. This can be passed into BaseApp.
func Stack(issuer weave.Address, nftBuckets map[string]orm.Bucket, minFee coin.Coin) weave.Handler {
	authFn := Authenticator()
	return Chain(authFn, minFee).WithHandler(Router(authFn, issuer, nftBuckets))
}

// Application constructs a basic ABCI application with
// the given arguments. If you are not sure what to use
// for the Handler, just use Stack().
func Application(name string, h weave.Handler,
	tx weave.TxDecoder, dbPath string, options *server.Options) (app.BaseApp, error) {

	ctx := context.Background()
	kv, err := CommitKVStore(dbPath)
	if err != nil {
		return app.BaseApp{}, err
	}
	RegisterNft()
	store := app.NewStoreApp(name, kv, QueryRouter(options.MinFee), ctx)
	base := app.NewBaseApp(store, tx, h, nil, options.Debug)
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
