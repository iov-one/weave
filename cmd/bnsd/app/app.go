/*
Package app links together all the various components
to construct the bnsd app.
*/
package bnsd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/cmd/bnsd/x/username"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/commands/server"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store/iavl"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/aswap"
	"github.com/iov-one/weave/x/batch"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/cron"
	"github.com/iov-one/weave/x/currency"
	"github.com/iov-one/weave/x/distribution"
	"github.com/iov-one/weave/x/escrow"
	"github.com/iov-one/weave/x/gov"
	"github.com/iov-one/weave/x/msgfee"
	"github.com/iov-one/weave/x/multisig"
	"github.com/iov-one/weave/x/sigs"
	"github.com/iov-one/weave/x/utils"
	"github.com/iov-one/weave/x/validators"
)

// Authenticator returns the typical authentication,
// just using public key signatures
func Authenticator() x.Authenticator {
	return x.ChainAuth(sigs.Authenticate{}, multisig.Authenticate{})
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
		batch.NewDecorator(),
		utils.NewActionTagger(),
	)
}

// ctrl can be initialized with any implementation, but must be used
// consistently everywhere.
var ctrl = cash.NewController(cash.NewBucket())

// Router returns a default router, only dispatching to the
// cash.SendMsg
func Router(authFn x.Authenticator, issuer weave.Address) *app.Router {
	r := app.NewRouter()
	scheduler := cron.NewScheduler(CronTaskMarshaler)

	migration.RegisterRoutes(r, authFn)
	cash.RegisterRoutes(r, authFn, ctrl)
	escrow.RegisterRoutes(r, authFn, ctrl)
	multisig.RegisterRoutes(r, authFn)
	//TODO: Possibly revisit passing the bucket later to have more control over types?
	// or implement a check
	currency.RegisterRoutes(r, authFn, issuer)
	validators.RegisterRoutes(r, authFn)
	distribution.RegisterRoutes(r, authFn, ctrl)
	sigs.RegisterRoutes(r, authFn)
	aswap.RegisterRoutes(r, authFn, ctrl)
	gov.RegisterRoutes(r, authFn, decodeProposalOptions, proposalOptionsExecutor(ctrl), scheduler)
	username.RegisterRoutes(r, authFn)
	return r
}

// QueryRouter returns a default query router.
func QueryRouter(minFee coin.Coin) weave.QueryRouter {
	r := weave.NewQueryRouter()
	antiSpamQuery := msgfee.NewAntiSpamQuery(minFee)

	r.RegisterAll(
		migration.RegisterQuery,
		escrow.RegisterQuery,
		cash.RegisterQuery,
		sigs.RegisterQuery,
		multisig.RegisterQuery,
		validators.RegisterQuery,
		orm.RegisterQuery,
		currency.RegisterQuery,
		distribution.RegisterQuery,
		antiSpamQuery.RegisterQuery,
		aswap.RegisterQuery,
		gov.RegisterQuery,
		username.RegisterQuery,
		cron.RegisterQuery,
	)
	return r
}

// Stack wires up a standard router with a standard decorator
// chain. This can be passed into BaseApp.
func Stack(issuer weave.Address, minFee coin.Coin) weave.Handler {
	authFn := Authenticator()
	return Chain(authFn, minFee).WithHandler(Router(authFn, issuer))
}

// CronStack wires up a standard router with a cron specific decorator chain.
// This can be passed into BaseApp.
// Cron stack configuration is a subset of the main stack. It is using the same
// components but not all functionalities are needed or expected (ie no message
// fee).
func CronStack() weave.Handler {
	rt := app.NewRouter()

	authFn := cron.Authenticator{}

	// Cron is using custom router as not the same handlers are registered.
	gov.RegisterCronRoutes(rt, authFn, decodeProposalOptions, proposalOptionsExecutor(ctrl))
	distribution.RegisterRoutes(rt, authFn, ctrl)
	escrow.RegisterRoutes(rt, authFn, ctrl)
	aswap.RegisterRoutes(rt, authFn, ctrl)

	decorators := app.ChainDecorators(
		utils.NewLogging(),
		utils.NewRecovery(),
		utils.NewKeyTagger(),
		utils.NewActionTagger(),
		// No fee decorators.
	)
	return decorators.WithHandler(rt)
}

// Application constructs a basic ABCI application with
// the given arguments. If you are not sure what to use
// for the Handler, just use Stack().
func Application(
	name string,
	h weave.Handler,
	tx weave.TxDecoder,
	dbPath string,
	options *server.Options,
) (app.BaseApp, error) {
	ctx := context.Background()
	kv, err := CommitKVStore(dbPath)
	if err != nil {
		return app.BaseApp{}, errors.Wrap(err, "cannot create store")
	}
	store := app.NewStoreApp(name, kv, QueryRouter(options.MinFee), ctx)
	ticker := cron.NewTicker(CronStack(), CronTaskMarshaler)
	base := app.NewBaseApp(store, tx, h, ticker, options.Debug)
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
