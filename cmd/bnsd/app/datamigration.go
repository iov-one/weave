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
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/msgfee"
)

func init() {
	datamigration.MustRegister("no-op test", datamigration.Migration{
		RequiredSigners: []weave.Address{devnetRule2},
		ChainIDs:        []string{"local-iov-devnet"},
		Migrate:         func(ctx context.Context, db weave.KVStore) error { return nil },
	})

	datamigration.MustRegister("initialize x/msgfee configuration owner", datamigration.Migration{
		RequiredSigners: []weave.Address{governingBoard},
		ChainIDs: []string{
			"iov-mainnet",
		},
		Migrate: initializeMsgfeeConfiguration,
	})
	datamigration.MustRegister("rewrite username accounts", datamigration.Migration{
		RequiredSigners: []weave.Address{governingBoard},
		ChainIDs: []string{
			"iov-mainnet",
		},
		Migrate: rewriteUsernameAccounts,
	})
	datamigration.MustRegister("initialize preregistration configuration", datamigration.Migration{
		RequiredSigners: []weave.Address{governingBoard},
		ChainIDs: []string{
			"iov-mainnet",
		},
		Migrate: initializePreregistrationConfiguration,
	})
	datamigration.MustRegister("rewrite preregistration records", datamigration.Migration{
		RequiredSigners: []weave.Address{governingBoard},
		ChainIDs: []string{
			"iov-mainnet",
		},
		Migrate: rewritePreregistrationRecords,
	})
}

var (
	governingBoard     = mustParse("cond:gov/rule/0000000000000001")
	devnetRule2        = mustParse("cond:gov/rule/0000000000000002")
	technicalExecutors = mustParse("cond:gov/rule/0000000000000003")
)

func mustParse(encodedAddress string) weave.Address {
	a, err := weave.ParseAddress(encodedAddress)
	if err != nil {
		panic(err)
	}
	return a
}

func initializePreregistrationConfiguration(ctx context.Context, db weave.KVStore) error {
	conf := preregistration.Configuration{
		Metadata: &weave.Metadata{Schema: 1},
		Owner:    technicalExecutors,
	}
	if err := gconf.Save(db, "preregistration", &conf); err != nil {
		return errors.Wrap(err, "save")
	}
	return nil
}

func initializeMsgfeeConfiguration(ctx context.Context, db weave.KVStore) error {
	var conf msgfee.Configuration
	switch err := gconf.Load(db, "msgfee", &msgfee.Configuration{}); {
	case errors.ErrNotFound.Is(err):
		conf.Metadata = &weave.Metadata{Schema: 1}
		conf.Owner = technicalExecutors
	case err == nil:
		if len(conf.Owner) != 0 {
			return errors.Wrap(errors.ErrState, "configuration owner already set")
		}
		conf.Owner = technicalExecutors
	default:
		return errors.Wrap(err, "load")
	}

	if err := gconf.Save(db, "msgfee", &conf); err != nil {
		return errors.Wrap(err, "save")
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
			name, domain := parseUsername(key)
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
	// Around 100 years.
	oneHundredYears = 100 * 365 * 24 * time.Hour
	// Around 10 years.
	tenYears = 10 * 365 * 24 * time.Hour
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
				AccountRenew: weave.AsUnixDuration(tenYears),
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
