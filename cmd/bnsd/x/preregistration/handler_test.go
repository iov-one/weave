package preregistration

import (
	"context"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

func TestHandlers(t *testing.T) {
	adminCond := weavetest.NewCondition()

	rt := app.NewRouter()
	auth := &weavetest.CtxAuth{Key: "auth"}
	RegisterRoutes(rt, auth)

	db := store.MemStore()
	migration.MustInitPkg(db, "preregistration")

	// Configuration must be initialized in order to authenticate
	// registration message.
	config := Configuration{
		Metadata: &weave.Metadata{Schema: 1},
		Owner:    adminCond.Address(),
	}
	if err := gconf.Save(db, "preregistration", &config); err != nil {
		t.Fatalf("cannot save configuration: %s", err)
	}

	ctx := weave.WithHeight(context.Background(), 1)
	ctx = auth.SetConditions(ctx, adminCond)
	ctx = weave.WithChainID(ctx, "testchain-123")
	ctx = weave.WithBlockTime(ctx, time.Now())

	tx := &weavetest.Tx{Msg: &RegisterMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Domain:   "stuff",
		Owner:    weavetest.NewCondition().Address(),
	}}
	cache := db.CacheWrap()
	if _, err := rt.Check(ctx, cache, tx); err != nil {
		t.Fatalf("check: %s", err)
	}
	cache.Discard()
	if _, err := rt.Deliver(ctx, db, tx); err != nil {
		t.Fatalf("deliver: %s", err)
	}

	b := NewRecordBucket()
	if err := b.Has(db, []byte("stuff")); err != nil {
		t.Fatalf("has: %s", err)
	}

	// The same domain cannot be registered twice.
	cache = db.CacheWrap()
	if _, err := rt.Check(ctx, cache, tx); !errors.ErrDuplicate.Is(err) {
		t.Fatalf("check: want duplicate, got %q", err)
	}
	cache.Discard()
	if _, err := rt.Deliver(ctx, db, tx); !errors.ErrDuplicate.Is(err) {
		t.Fatalf("deliver: want duplicate, got %q", err)
	}
}
