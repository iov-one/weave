package bnsd

import (
	"context"
	"strings"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/x/account"
	"github.com/iov-one/weave/cmd/bnsd/x/username"
	"github.com/iov-one/weave/datamigration"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/msgfee"
)

func init() {
	technicalExecutors, err := weave.ParseAddress("cond:gov/rule/0000000000000003")
	if err != nil {
		panic(err)
	}
	governingBoard, err := weave.ParseAddress("cond:gov/rule/0000000000000001")
	if err != nil {
		panic(err)
	}

	datamigration.MustRegister("initialize x/msgfee configuration owner", datamigration.Migration{
		RequiredSigners: []weave.Address{governingBoard},
		ChainIDs: []string{
			"iov-mainnet",
		},
		Migrate: func(ctx context.Context, db weave.KVStore) error {
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
		},
	})

	datamigration.MustRegister("rewrite username accounts", datamigration.Migration{
		RequiredSigners: []weave.Address{governingBoard},
		ChainIDs: []string{
			"iov-mainnet",
		},
		Migrate: rewriteUsernameAccounts,
	})
}

func rewriteUsernameAccounts(ctx context.Context, db weave.KVStore) error {
	governingBoard, err := weave.ParseAddress("cond:gov/rule/0000000000000001")
	if err != nil {
		panic(err)
	}

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
