package username

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x"
)

func TestRegisterTokenHandler(t *testing.T) {
	var (
		aliceCond = weavetest.NewCondition()
		bobbyCond = weavetest.NewCondition()
	)

	cases := map[string]struct {
		Init           func(t testing.TB, db weave.KVStore)
		Tx             weave.Tx
		Auth           x.Authenticator
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
	}{
		"success": {
			Tx: &weavetest.Tx{
				Msg: &RegisterTokenMsg{
					Username: "bobby@iov",
					Targets: []BlockchainAddress{
						{BlockchainID: "bc_1", Address: []byte("addr1")},
						{BlockchainID: "bc_2", Address: []byte("addr2")},
					},
				},
			},
			Auth: &weavetest.Auth{Signer: bobbyCond},
		},
		"username must be unique": {
			Tx: &weavetest.Tx{
				Msg: &RegisterTokenMsg{
					Username: "alice@iov",
					Targets: []BlockchainAddress{
						{BlockchainID: "bc_1", Address: []byte("addr1")},
					},
				},
			},
			Auth:           &weavetest.Auth{Signer: aliceCond},
			WantCheckErr:   errors.ErrDuplicate,
			WantDeliverErr: errors.ErrDuplicate,
		},
		"target cannot be empty": {
			Tx: &weavetest.Tx{
				Msg: &RegisterTokenMsg{
					Username: "alice@iov",
					Targets:  []BlockchainAddress{},
				},
			},
			Auth:           &weavetest.Auth{Signer: aliceCond},
			WantCheckErr:   errors.ErrEmpty,
			WantDeliverErr: errors.ErrEmpty,
		},
		"username must be provided": {
			Tx: &weavetest.Tx{
				Msg: &RegisterTokenMsg{
					Username: "",
					Targets: []BlockchainAddress{
						{BlockchainID: "bc_1", Address: []byte("addr1")},
					},
				},
			},
			Auth:           &weavetest.Auth{Signer: aliceCond},
			WantCheckErr:   errors.ErrEmpty,
			WantDeliverErr: errors.ErrEmpty,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, "username")

			b := NewTokenBucket()
			_, err := b.Put(db, []byte("alice*iov"), &Token{
				Metadata: &weave.Metadata{Schema: 1},
				Targets: []BlockchainAddress{
					{BlockchainID: "unichain", Address: []byte("756e69636f696e2d310a")},
				},
				Owner: aliceCond.Address(),
			})
			assert.Nil(t, err)

			h := registerTokenHandler{
				auth:   tc.Auth,
				bucket: b,
			}

			cache := db.CacheWrap()
			if _, err := h.Check(context.TODO(), cache, tc.Tx); tc.WantCheckErr.Is(err) {
				t.Fatalf("unexpected check error: %s", err)
			}
			cache.Discard()
			if _, err := h.Deliver(context.TODO(), db, tc.Tx); tc.WantCheckErr.Is(err) {
				t.Fatalf("unexpected deliver error: %s", err)
			}
		})
	}
}

func TestChangeTokenOwnerHandler(t *testing.T) {
	var (
		aliceCond = weavetest.NewCondition()
		bobbyCond = weavetest.NewCondition()
	)

	cases := map[string]struct {
		Tx             weave.Tx
		Auth           x.Authenticator
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
	}{
		"success": {
			Tx: &weavetest.Tx{
				Msg: &ChangeTokenOwnerMsg{
					Username: "alice@iov",
					NewOwner: bobbyCond.Address(),
				},
			},
			Auth: &weavetest.Auth{Signer: aliceCond},
		},
		"only the owner can change the token": {
			Tx: &weavetest.Tx{
				Msg: &ChangeTokenOwnerMsg{
					Username: "alice@iov",
					NewOwner: bobbyCond.Address(),
				},
			},
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
			Auth:           &weavetest.Auth{Signer: bobbyCond},
		},
		"token must exist": {
			Tx: &weavetest.Tx{
				Msg: &ChangeTokenOwnerMsg{
					Username: "does-not-exist@iov",
					NewOwner: bobbyCond.Address(),
				},
			},
			WantCheckErr:   errors.ErrNotFound,
			WantDeliverErr: errors.ErrNotFound,
			Auth:           &weavetest.Auth{Signer: bobbyCond},
		},
	}
	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, "username")

			b := NewTokenBucket()
			_, err := b.Put(db, []byte("alice*iov"), &Token{
				Metadata: &weave.Metadata{Schema: 1},
				Targets: []BlockchainAddress{
					{BlockchainID: "unichain", Address: []byte("756e69636f696e2d310a")},
				},
				Owner: aliceCond.Address(),
			})
			assert.Nil(t, err)

			h := changeTokenOwnerHandler{
				auth:   tc.Auth,
				bucket: b,
			}

			cache := db.CacheWrap()
			if _, err := h.Check(context.TODO(), cache, tc.Tx); tc.WantCheckErr.Is(err) {
				t.Fatalf("unexpected check error: %s", err)
			}
			cache.Discard()
			if _, err := h.Deliver(context.TODO(), db, tc.Tx); tc.WantCheckErr.Is(err) {
				t.Fatalf("unexpected deliver error: %s", err)
			}
		})
	}
}

func TestChangeTokenTargetHandler(t *testing.T) {
	var (
		aliceCond = weavetest.NewCondition()
		bobbyCond = weavetest.NewCondition()
	)
	cases := map[string]struct {
		Tx             weave.Tx
		Auth           x.Authenticator
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
	}{
		"success": {
			Tx: &weavetest.Tx{
				Msg: &ChangeTokenTargetsMsg{
					Username: "alice@iov",
					NewTargets: []BlockchainAddress{
						{BlockchainID: "hydracoin", Address: []byte("68796472612d310a")},
						{BlockchainID: "pegauscoin", Address: []byte("706567617375732d310a")},
					},
				},
			},
			Auth: &weavetest.Auth{Signer: aliceCond},
		},
		"only the owner can change the token": {
			Tx: &weavetest.Tx{
				Msg: &ChangeTokenTargetsMsg{
					Username: "alice@iov",
					NewTargets: []BlockchainAddress{
						{BlockchainID: "hydracoin", Address: []byte("68796472612d310a")},
					},
				},
			},
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
			Auth:           &weavetest.Auth{Signer: bobbyCond},
		},
		"token must exist": {
			Tx: &weavetest.Tx{
				Msg: &ChangeTokenTargetsMsg{
					Username: "does-not-exist@iov",
					NewTargets: []BlockchainAddress{
						{BlockchainID: "hydracoin", Address: []byte("68796472612d310a")},
					},
				},
			},
			WantCheckErr:   errors.ErrNotFound,
			WantDeliverErr: errors.ErrNotFound,
			Auth:           &weavetest.Auth{Signer: bobbyCond},
		},
	}
	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, "username")

			b := NewTokenBucket()
			_, err := b.Put(db, []byte("alice*iov"), &Token{
				Metadata: &weave.Metadata{Schema: 1},
				Targets: []BlockchainAddress{
					{BlockchainID: "unichain", Address: []byte("756e69636f696e2d310a")},
				},
				Owner: aliceCond.Address(),
			})
			assert.Nil(t, err)

			h := changeTokenTargetsHandler{
				auth:   tc.Auth,
				bucket: b,
			}

			cache := db.CacheWrap()
			if _, err := h.Check(context.TODO(), cache, tc.Tx); tc.WantCheckErr.Is(err) {
				t.Fatalf("unexpected check error: %s", err)
			}
			cache.Discard()
			if _, err := h.Deliver(context.TODO(), db, tc.Tx); tc.WantCheckErr.Is(err) {
				t.Fatalf("unexpected deliver error: %s", err)
			}
		})
	}
}
