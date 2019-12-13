package bnsd

import (
	"context"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/datamigration"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
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
		ChainID:         "iov-mainnet",
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
}
