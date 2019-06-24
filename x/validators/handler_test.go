package validators

import (
	"context"
	"reflect"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

func TestHandler(t *testing.T) {
	alice := weavetest.NewKey()
	bobby := weavetest.NewKey()

	specs := map[string]struct {
		Initial       weave.ValidatorUpdates
		Src           []weave.ValidatorUpdate
		AuthzAddress  weave.Address
		ExpCheckErr   *errors.Error
		ExpDeliverErr *errors.Error
		Exp           []weave.ValidatorUpdate
		DbExp         weave.ValidatorUpdates
	}{
		"All good with authorized address": {
			Initial: weave.ValidatorUpdates{ValidatorUpdates: []weave.ValidatorUpdate{{PubKey: weave.PubKey{Data: bobby.PublicKey().GetEd25519(), Type: "ed25519"}, Power: 3}}},
			Src: []weave.ValidatorUpdate{{
				PubKey: weave.PubKey{Data: alice.PublicKey().GetEd25519(), Type: "ed25519"},
				Power:  10,
			}},
			AuthzAddress: alice.PublicKey().Address(),
			Exp: []weave.ValidatorUpdate{{
				PubKey: weave.PubKey{Data: alice.PublicKey().GetEd25519(), Type: "ed25519"},
				Power:  10,
			}},
			DbExp: weave.ValidatorUpdates{ValidatorUpdates: []weave.ValidatorUpdate{{PubKey: weave.PubKey{Data: bobby.PublicKey().GetEd25519(), Type: "ed25519"}, Power: 3}, {
				PubKey: weave.PubKey{Data: alice.PublicKey().GetEd25519(), Type: "ed25519"},
				Power:  10,
			}}},
		},
		"Adding a validator works with pre-exiting one": {
			Src: []weave.ValidatorUpdate{{
				PubKey: weave.PubKey{Data: alice.PublicKey().GetEd25519(), Type: "ed25519"},
				Power:  10,
			}},
			AuthzAddress: alice.PublicKey().Address(),
			Exp: []weave.ValidatorUpdate{{
				PubKey: weave.PubKey{Data: alice.PublicKey().GetEd25519(), Type: "ed25519"},
				Power:  10,
			}},
			DbExp: weave.ValidatorUpdates{ValidatorUpdates: []weave.ValidatorUpdate{{
				PubKey: weave.PubKey{Data: alice.PublicKey().GetEd25519(), Type: "ed25519"},
				Power:  10,
			}}},
		},
		"Setting different power is allowed": {
			Initial: weave.ValidatorUpdates{ValidatorUpdates: []weave.ValidatorUpdate{{PubKey: weave.PubKey{Data: alice.PublicKey().GetEd25519(), Type: "ed25519"}, Power: 3}}},
			Src: []weave.ValidatorUpdate{{
				PubKey: weave.PubKey{Data: alice.PublicKey().GetEd25519(), Type: "ed25519"},
				Power:  1,
			}},
			AuthzAddress: alice.PublicKey().Address(),
			Exp: []weave.ValidatorUpdate{{
				PubKey: weave.PubKey{Data: alice.PublicKey().GetEd25519(), Type: "ed25519"},
				Power:  1,
			}},
			DbExp: weave.ValidatorUpdates{ValidatorUpdates: []weave.ValidatorUpdate{{
				PubKey: weave.PubKey{Data: alice.PublicKey().GetEd25519(), Type: "ed25519"},
				Power:  1,
			}}},
		},
		"Power 0 is allowed to remove a validator": {
			Initial: weave.ValidatorUpdates{ValidatorUpdates: []weave.ValidatorUpdate{{PubKey: weave.PubKey{Data: alice.PublicKey().GetEd25519(), Type: "ed25519"}, Power: 1}}},
			Src: []weave.ValidatorUpdate{{
				PubKey: weave.PubKey{Data: alice.PublicKey().GetEd25519(), Type: "ed25519"},
				Power:  0,
			}},
			AuthzAddress: alice.PublicKey().Address(),
			Exp: []weave.ValidatorUpdate{{
				PubKey: weave.PubKey{Data: alice.PublicKey().GetEd25519(), Type: "ed25519"},
				Power:  0,
			}},
		},
		"Power 0 is fails if the validator does not exist": {
			Src: []weave.ValidatorUpdate{{
				PubKey: weave.PubKey{Data: alice.PublicKey().GetEd25519(), Type: "ed25519"},
				Power:  0,
			}},
			AuthzAddress:  alice.PublicKey().Address(),
			ExpCheckErr:   errors.ErrInput,
			ExpDeliverErr: errors.ErrInput,
		},
		"Negative power prohibited": {
			Src: []weave.ValidatorUpdate{{
				PubKey: weave.PubKey{Data: alice.PublicKey().GetEd25519(), Type: "ed25519"},
				Power:  -1,
			}},
			AuthzAddress:  alice.PublicKey().Address(),
			ExpCheckErr:   errors.ErrMsg,
			ExpDeliverErr: errors.ErrMsg,
		},
		"Invalid public key": {
			Src: []weave.ValidatorUpdate{{
				PubKey: weave.PubKey{Data: []byte{0, 1, 2}, Type: "ed25519"},
				Power:  10,
			}},
			AuthzAddress:  alice.PublicKey().Address(),
			ExpCheckErr:   errors.ErrType,
			ExpDeliverErr: errors.ErrType,
		},
		"Empty validator set prohibited": {
			Src:           []weave.ValidatorUpdate{},
			AuthzAddress:  alice.PublicKey().Address(),
			ExpCheckErr:   errors.ErrEmpty,
			ExpDeliverErr: errors.ErrEmpty,
		},
		"Unauthorized address should fail": {
			Src: []weave.ValidatorUpdate{{
				PubKey: weave.PubKey{Data: alice.PublicKey().GetEd25519(), Type: "ed25519"},
				Power:  10,
			}},
			AuthzAddress:  bobby.PublicKey().Address(),
			ExpCheckErr:   errors.ErrUnauthorized,
			ExpDeliverErr: errors.ErrUnauthorized,
		},
	}

	auth := &weavetest.Auth{
		Signer: alice.PublicKey().Condition(),
	}
	rt := app.NewRouter()
	RegisterRoutes(rt, auth)

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, "validators")
			ctx := context.Background()
			err := NewAccountBucket().Save(db, AccountsWith(WeaveAccounts{Addresses: []weave.Address{spec.AuthzAddress}}))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if err := weave.StoreValidatorUpdates(db, spec.Initial); err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			cache := db.CacheWrap()
			tx := &weavetest.Tx{Msg: &ApplyDiffMsg{
				Metadata:         &weave.Metadata{Schema: 1},
				ValidatorUpdates: spec.Src,
			}}
			// when check is called
			if _, err := rt.Check(ctx, cache, tx); !spec.ExpCheckErr.Is(err) {
				t.Fatalf("check expected: %+v  but got %+v", spec.ExpCheckErr, err)
			}
			cache.Discard()

			// and when deliver is called
			res, err := rt.Deliver(ctx, db, tx)
			if !spec.ExpDeliverErr.Is(err) {
				t.Fatalf("deliver expected: %+v  but got %+v", spec.ExpCheckErr, err)
			}
			if spec.ExpDeliverErr != nil {
				return // skip further checks on expected error
			}
			if exp, got := spec.Exp, res.Diff; !reflect.DeepEqual(exp, got) {
				t.Errorf("expected %v but got %v", exp, got)
			}
			got, err := weave.GetValidatorUpdates(db)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !reflect.DeepEqual(spec.DbExp, got) {
				t.Errorf("expected %v but got %v", spec.Exp, got)
			}

		})
	}
}
