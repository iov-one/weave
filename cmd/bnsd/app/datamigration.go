package bnsd

import (
	"context"
	"strings"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/x/account"
	"github.com/iov-one/weave/cmd/bnsd/x/preregistration"
	"github.com/iov-one/weave/cmd/bnsd/x/username"
	"github.com/iov-one/weave/datamigration"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	datamigration.MustRegister("no-op test", datamigration.Migration{
		RequiredSigners: []weave.Address{economicExecutors},
		ChainIDs: []string{
			"iov-dancenet",
			"local-iov-devnet",
		},
		Migrate: func(ctx context.Context, db weave.KVStore) error { return nil },
	})

	datamigration.MustRegister("version 1.0 release", datamigration.Migration{
		RequiredSigners: []weave.Address{governingBoard},
		ChainIDs: []string{
			"iov-dancenet",
			"iov-mainnet",
		},
		Migrate: migrateRelease_1_0,
	})
}

var (
	governingBoard     = mustParse("cond:gov/rule/0000000000000001")
	economicExecutors  = mustParse("cond:gov/rule/0000000000000002")
	technicalExecutors = mustParse("cond:gov/rule/0000000000000003")
)

func mustParse(encodedAddress string) weave.Address {
	a, err := weave.ParseAddress(encodedAddress)
	if err != nil {
		panic(err)
	}
	return a
}

// migrateRelease_1_0 clubs together several migrations required for the 1.0
// release. Because they are running within a single migration execution,
// atomic execution is guaranteed.
func migrateRelease_1_0(ctx context.Context, db weave.KVStore) error {
	if err := initializeSchema(db, "account"); err != nil {
		return errors.Wrap(err, "initialize account schema")
	}
	if err := gconf.Save(db, "account", &account.Configuration{
		Metadata:               &weave.Metadata{Schema: 1},
		Owner:                  technicalExecutors,
		ValidDomain:            `^[a-z0-9]+$`,
		ValidName:              `^[a-z0-9\-_.]{3,64}$`,
		ValidBlockchainID:      `^[a-z0-9A-Z\-]+$`,
		ValidBlockchainAddress: `^[a-z0-9A-Z]+$`,
		DomainRenew:            weave.AsUnixDuration(365*24*time.Hour + 6*time.Hour),
		DomainGracePeriod:      weave.AsUnixDuration(30 * 24 * time.Hour),
	}); err != nil {
		return errors.Wrap(err, "save initial gconf configuration")
	}
	if err := rewriteUsernameAccounts(ctx, db); err != nil {
		return errors.Wrap(err, "rewrite username accounts")
	}
	if err := rewritePreregistrationRecords(ctx, db); err != nil {
		return errors.Wrap(err, "rewrite preregistration records")
	}
	if err := rewriteAccountBlockchainIDs(ctx, db); err != nil {
		return errors.Wrap(err, "rewrite account blockchain ID")
	}
	if err := gconf.Save(db, "account", &account.Configuration{
		Metadata:               &weave.Metadata{Schema: 1},
		Owner:                  technicalExecutors,
		ValidDomain:            `^[a-z0-9\-_]{3,16}$`,
		ValidName:              `^[a-z0-9\-_.]{3,64}$`,
		ValidBlockchainID:      `^[a-z0-9A-Z\-:]+$`,
		ValidBlockchainAddress: `^[a-z0-9A-Z]+$`,
		DomainRenew:            weave.AsUnixDuration(365*24*time.Hour + 6*time.Hour),
		DomainGracePeriod:      weave.AsUnixDuration(30 * 24 * time.Hour),
	}); err != nil {
		return errors.Wrap(err, "save final gconf configuration")
	}
	return nil
}

// initializeSchema register a schema information with version 1 for the given
// package name (extension). This function fails if schema for requested
// extension was already registered.
func initializeSchema(db weave.KVStore, pkgName string) error {
	b := migration.NewSchemaBucket()
	switch ver, err := b.CurrentSchema(db, pkgName); {
	case err == nil:
		return errors.Wrapf(errors.ErrSchema, "initialized with version %d", ver)
	case errors.ErrNotFound.Is(err):
		schema := migration.Schema{
			Metadata: &weave.Metadata{Schema: 1},
			Pkg:      pkgName,
			Version:  1,
		}
		if _, err := b.Create(db, &schema); err != nil {
			return errors.Wrap(err, "create schema information")
		}
	default:
		return errors.Wrap(err, "current schema version")
	}
	return nil
}

func rewriteUsernameAccounts(ctx context.Context, db weave.KVStore) error {
	now, err := weave.BlockTime(ctx)
	if err != nil {
		return errors.Wrap(err, "block time")
	}

	iov := account.Domain{
		Metadata:     &weave.Metadata{Schema: 1},
		Domain:       "iov",
		Admin:        governingBoard,
		HasSuperuser: false,
		AccountRenew: weave.AsUnixDuration(tenYears),
		// IOV domain is not supposed to expire. It is
		// not our problem in 100 years ;)
		ValidUntil: weave.AsUnixTime(now.Add(oneHundredYears)),
	}
	if _, err := account.NewDomainBucket().Put(db, []byte("iov"), &iov); err != nil {
		return errors.Wrap(err, "save iov domain")
	}

	accounts := account.NewAccountBucket()

	// Every domain must contain an empty account.
	empty := &account.Account{
		Metadata:     &weave.Metadata{Schema: 1},
		Domain:       "iov",
		Name:         "",
		Owner:        iov.Admin,
		ValidUntil:   weave.AsUnixTime(now.Add(iov.AccountRenew.Duration())),
		Certificates: nil,
	}
	if _, err := accounts.Put(db, []byte("*iov"), empty); err != nil {
		return errors.Wrap(err, "save empty account")
	}

	it := orm.IterAll("tokens")
	for {
		var token username.Token
		switch key, err := it.Next(db, &token); {
		case err == nil:
			name, domain := parseUsername(string(key))
			if domain != "iov" {
				// Ignore all non IOV domains. Username should not contain
				// any non IOV names, but better be sure.
				continue
			}
			acc := &account.Account{
				Metadata:     &weave.Metadata{Schema: 1},
				Domain:       "iov",
				Name:         name,
				Owner:        token.Owner,
				ValidUntil:   weave.AsUnixTime(now.Add(iov.AccountRenew.Duration())),
				Certificates: nil,
			}
			for _, t := range token.Targets {
				acc.Targets = append(acc.Targets, account.BlockchainAddress{
					BlockchainID: t.BlockchainID,
					Address:      t.Address,
				})
			}
			accountKey := []byte(name + "*" + domain)
			if _, err := accounts.Put(db, accountKey, acc); err != nil {
				return errors.Wrapf(err, "save account %q", key)
			}
		case errors.ErrIteratorDone.Is(err):
			return nil
		default:
			return errors.Wrap(err, "iterator next")
		}
	}
}

func parseUsername(u string) (string, string) {
	chunks := strings.SplitN(u, "*", 2)
	return chunks[0], chunks[1]
}

const (
	// Below durations are good enough estimations.
	oneYear         = 365 * 24 * time.Hour
	tenYears        = 10 * oneYear
	oneHundredYears = 100 * oneYear
)

func rewritePreregistrationRecords(ctx context.Context, db weave.KVStore) error {
	now, err := weave.BlockTime(ctx)
	if err != nil {
		return errors.Wrap(err, "block time")
	}

	domains := account.NewDomainBucket()

	var conf account.Configuration
	if err := gconf.Load(db, "account", &conf); err != nil {
		return errors.Wrap(err, "load account configuration")
	}

	it := orm.IterAll("records")
	for {
		var record preregistration.Record
		switch _, err := it.Next(db, &record); {
		case err == nil:
			domain := account.Domain{
				Metadata:     &weave.Metadata{Schema: 1},
				Domain:       record.Domain,
				Admin:        record.Owner,
				HasSuperuser: true,
				AccountRenew: weave.AsUnixDuration(oneYear),
				ValidUntil:   weave.AsUnixTime(now.Add(conf.DomainRenew.Duration())),
			}
			if _, err := domains.Put(db, []byte(record.Domain), &domain); err != nil {
				return errors.Wrapf(err, "save %q domain", record.Domain)
			}
		case errors.ErrIteratorDone.Is(err):
			return nil
		default:
			return errors.Wrap(err, "iterator next")
		}
	}
}

func rewriteAccountBlockchainIDs(ctx context.Context, db weave.KVStore) error {
	b := account.NewAccountBucket()
	it := orm.IterAll("account")
	for {
		var ac account.Account
		switch key, err := it.Next(db, &ac); {
		case err == nil:
			targets, changed := migrateAccountTargetBlockchainID(ac.Targets)
			if !changed {
				continue
			}
			ac.Targets = targets
			if _, err := b.Put(db, key, &ac); err != nil {
				return errors.Wrapf(err, "cannot save %q account", key)
			}
		case errors.ErrIteratorDone.Is(err):
			return nil
		default:
			return errors.Wrap(err, "iterator next")
		}
	}
}

// migrateAccountTargetBlockchainID updates any BlockchainID to CAIP specified
// if possible.
// See https://github.com/ChainAgnostic/CAIPs/tree/master/CAIPs
func migrateAccountTargetBlockchainID(targets []account.BlockchainAddress) ([]account.BlockchainAddress, bool) {
	var updated bool
	// this is a set of old chain ID to new chain ID
	var oldToNewChainID = map[string]string{
		"ethereum-eip155-1":           "eip155:1",
		"iov-mainnet":                 "cosmos:iov-mainnet",
		"lisk-ed14889723":             "lip9:9ee11e9df416b18b",
		"cosmos-binance-chain-tigris": "cosmos:Binance-Chain-Tigris",
		"bip122-tmp-bitcoin":          "bip122:000000000019d6689c085ae165831e93",
		"bip122-tmp-bcash":            "bip122:000000000000000000651ef99cb9fcbe",
		"cosmos-cosmoshub-3":          "cosmos:cosmoshub-3",
		"cosmos-irishub":              "cosmos:irishub",
		"cosmos-kava-2":               "cosmos:kava-2",
		"bip122-tmp-litecoin":         "bip122:12a765e31ffd4059bada1e25190f6e98",
		"cosmos-columbus-3":           "cosmos:columbus-3",
		"tezos-tmp-mainnet":           "tezos:NetXdQprcVkpaWU",
		"cosmos-emoney-1":             "cosmos:emoney-1",
	}
	// iterate over the accounts
	for i, t := range targets {
		// check if blockchain ID is in the set of IDs
		// that require to be updated
		updatedID, ok := oldToNewChainID[t.BlockchainID]
		// if the ID is not found in the ones we need to update
		// then continue iterating over blockchain addresses
		if !ok {
			continue
		}
		// otherwise update the blockchain ID
		targets[i].BlockchainID = updatedID
		updated = true
	}
	return targets, updated
}
