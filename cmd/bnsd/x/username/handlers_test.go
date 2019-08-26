package username

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x"
)

func TestRegisterNamespaceHandler(t *testing.T) {
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
				Msg: &RegisterNamespaceMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Label:    "evilcorp",
					Public:   true,
				},
			},
			Auth: &weavetest.Auth{Signer: bobbyCond},
		},
		"namespace with the same label cannot be registered twice": {
			Tx: &weavetest.Tx{
				Msg: &RegisterNamespaceMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Label:    "iov",
					Public:   true,
				},
			},
			Auth:           &weavetest.Auth{Signer: bobbyCond},
			WantCheckErr:   errors.ErrDuplicate,
			WantDeliverErr: errors.ErrDuplicate,
		},
		"namespace label must pass validation rules check": {
			Tx: &weavetest.Tx{
				Msg: &RegisterNamespaceMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Label:    "x", // Does not match gconf rules.
					Public:   true,
				},
			},
			Auth:           &weavetest.Auth{Signer: bobbyCond},
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, "username")

			config := Configuration{
				ValidUsernameName:  `[a-z0-9\-_.]{3,64}`,
				ValidUsernameLabel: `[a-z0-9]{3,16}`,
			}
			if err := gconf.Save(db, "username", &config); err != nil {
				t.Fatalf("cannot save configuration: %s", err)
			}

			namespaces := NewNamespaceBucket()

			// Preregister a namespace for each test.
			_, err := namespaces.Put(db, []byte("iov"), &Namespace{
				Metadata: &weave.Metadata{Schema: 1},
				Owner:    aliceCond.Address(),
				Public:   true,
			})
			assert.Nil(t, err)

			h := registerNamespaceHandler{
				auth:       tc.Auth,
				namespaces: namespaces,
			}

			cache := db.CacheWrap()
			if _, err := h.Check(context.TODO(), cache, tc.Tx); !tc.WantCheckErr.Is(err) {
				t.Fatalf("unexpected check error: %s", err)
			}
			cache.Discard()
			if _, err := h.Deliver(context.TODO(), db, tc.Tx); !tc.WantDeliverErr.Is(err) {
				t.Fatalf("unexpected deliver error: %s", err)
			}
		})
	}
}

func TestRegisterTokenHandler(t *testing.T) {
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
				Msg: &RegisterTokenMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Username: "bobby*iov",
					Targets: []BlockchainAddress{
						{BlockchainID: "bc_1", Address: "addr1"},
						{BlockchainID: "bc_2", Address: "addr2"},
					},
				},
			},
			Auth: &weavetest.Auth{Signer: bobbyCond},
		},
		"username must be unique": {
			Tx: &weavetest.Tx{
				Msg: &RegisterTokenMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Username: "alice*iov",
					Targets: []BlockchainAddress{
						{BlockchainID: "bc_1", Address: "addr1"},
					},
				},
			},
			Auth:           &weavetest.Auth{Signer: aliceCond},
			WantCheckErr:   errors.ErrDuplicate,
			WantDeliverErr: errors.ErrDuplicate,
		},
		"target can be empty": {
			Tx: &weavetest.Tx{
				Msg: &RegisterTokenMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Username: "alice2*iov",
					Targets:  []BlockchainAddress{},
				},
			},
			Auth: &weavetest.Auth{Signer: aliceCond},
		},
		"username must be provided": {
			Tx: &weavetest.Tx{
				Msg: &RegisterTokenMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Username: "",
					Targets: []BlockchainAddress{
						{BlockchainID: "bc_1", Address: "addr1"},
					},
				},
			},
			Auth:           &weavetest.Auth{Signer: aliceCond},
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"username must be registered": {
			Tx: &weavetest.Tx{
				Msg: &RegisterTokenMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Username: "bobby*fakecorp",
					Targets: []BlockchainAddress{
						{BlockchainID: "bc_1", Address: "addr1"},
					},
				},
			},
			Auth:           &weavetest.Auth{Signer: bobbyCond},
			WantCheckErr:   errors.ErrNotFound,
			WantDeliverErr: errors.ErrNotFound,
		},
		"anyone can registerd in a public namespace": {
			Tx: &weavetest.Tx{
				Msg: &RegisterTokenMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Username: "bobby*iov",
					Targets: []BlockchainAddress{
						{BlockchainID: "bc_1", Address: "addr1"},
					},
				},
			},
			Auth:           &weavetest.Auth{Signer: bobbyCond},
			WantCheckErr:   nil,
			WantDeliverErr: nil,
		},
		"not everyone can register in a private namespace": {
			Tx: &weavetest.Tx{
				Msg: &RegisterTokenMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Username: "bobby*privatecorp",
					Targets: []BlockchainAddress{
						{BlockchainID: "bc_1", Address: "addr1"},
					},
				},
			},
			Auth:           &weavetest.Auth{Signer: bobbyCond},
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
		},
		"the namespace owner can register a username in a private namespace": {
			Tx: &weavetest.Tx{
				Msg: &RegisterTokenMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Username: "alice*privatecorp",
					Targets: []BlockchainAddress{
						{BlockchainID: "bc_1", Address: "addr1"},
					},
				},
			},
			Auth:           &weavetest.Auth{Signer: aliceCond},
			WantCheckErr:   nil,
			WantDeliverErr: nil,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, "username")

			config := Configuration{
				ValidUsernameName:  `[a-z0-9\-_.]{3,64}`,
				ValidUsernameLabel: `[a-z0-9]{3,16}`,
			}
			if err := gconf.Save(db, "username", &config); err != nil {
				t.Fatalf("cannot save configuration: %s", err)
			}

			namespaces := NewNamespaceBucket()
			// Preregister two namespaces.
			_, err := namespaces.Put(db, []byte("iov"), &Namespace{
				Metadata: &weave.Metadata{Schema: 1},
				Owner:    aliceCond.Address(),
				Public:   true,
			})
			assert.Nil(t, err)
			_, err = namespaces.Put(db, []byte("privatecorp"), &Namespace{
				Metadata: &weave.Metadata{Schema: 1},
				Owner:    aliceCond.Address(),
				Public:   false,
			})
			assert.Nil(t, err)

			tokens := NewTokenBucket()
			_, err = tokens.Put(db, []byte("alice*iov"), &Token{
				Metadata: &weave.Metadata{Schema: 1},
				Targets: []BlockchainAddress{
					{BlockchainID: "unichain", Address: "some-unichain-address"},
				},
				Owner: aliceCond.Address(),
			})
			assert.Nil(t, err)

			h := registerTokenHandler{
				auth:       tc.Auth,
				tokens:     tokens,
				namespaces: namespaces,
			}

			cache := db.CacheWrap()
			if _, err := h.Check(context.TODO(), cache, tc.Tx); !tc.WantCheckErr.Is(err) {
				t.Fatalf("unexpected check error: %s", err)
			}
			cache.Discard()
			if _, err := h.Deliver(context.TODO(), db, tc.Tx); !tc.WantDeliverErr.Is(err) {
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
				Msg: &TransferTokenMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Username: "alice*iov",
					NewOwner: bobbyCond.Address(),
				},
			},
			Auth: &weavetest.Auth{Signer: aliceCond},
		},
		"only the owner can change the token": {
			Tx: &weavetest.Tx{
				Msg: &TransferTokenMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Username: "alice*iov",
					NewOwner: bobbyCond.Address(),
				},
			},
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
			Auth:           &weavetest.Auth{Signer: bobbyCond},
		},
		"token must exist": {
			Tx: &weavetest.Tx{
				Msg: &TransferTokenMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Username: "does-not-exist*iov",
					NewOwner: bobbyCond.Address(),
				},
			},
			WantCheckErr:   errors.ErrNotFound,
			WantDeliverErr: errors.ErrNotFound,
			Auth:           &weavetest.Auth{Signer: bobbyCond},
		},
		"change to the same owner (no change) is allowed": {
			Tx: &weavetest.Tx{
				Msg: &TransferTokenMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Username: "alice*iov",
					NewOwner: aliceCond.Address(),
				},
			},
			Auth: &weavetest.Auth{Signer: aliceCond},
		},
	}
	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, "username")

			tokens := NewTokenBucket()
			_, err := tokens.Put(db, []byte("alice*iov"), &Token{
				Metadata: &weave.Metadata{Schema: 1},
				Targets: []BlockchainAddress{
					{BlockchainID: "unichain", Address: "some-unichain-address"},
				},
				Owner: aliceCond.Address(),
			})
			assert.Nil(t, err)

			h := transferTokenHandler{
				auth:   tc.Auth,
				tokens: tokens,
			}

			cache := db.CacheWrap()
			if _, err := h.Check(context.TODO(), cache, tc.Tx); !tc.WantCheckErr.Is(err) {
				t.Fatalf("unexpected check error: %s", err)
			}
			cache.Discard()
			if _, err := h.Deliver(context.TODO(), db, tc.Tx); !tc.WantDeliverErr.Is(err) {
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
					Metadata: &weave.Metadata{Schema: 1},
					Username: "alice*iov",
					NewTargets: []BlockchainAddress{
						{BlockchainID: "hydracoin", Address: "some-hydra-address"},
						{BlockchainID: "pegasuscoin", Address: "some-pagasus-address"},
					},
				},
			},
			Auth: &weavetest.Auth{Signer: aliceCond},
		},
		"can change target to the same value (no change)": {
			Tx: &weavetest.Tx{
				Msg: &ChangeTokenTargetsMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Username: "alice*iov",
					NewTargets: []BlockchainAddress{
						{BlockchainID: "unichain", Address: "some-unicorn-address"},
					},
				},
			},
			Auth: &weavetest.Auth{Signer: aliceCond},
		},
		"invalid message": {
			Tx: &weavetest.Tx{
				Msg: &ChangeTokenTargetsMsg{
					Metadata:   nil,
					Username:   "",
					NewTargets: []BlockchainAddress{},
				},
			},
			Auth:           &weavetest.Auth{Signer: aliceCond},
			WantCheckErr:   errors.ErrMetadata,
			WantDeliverErr: errors.ErrMetadata,
		},
		"invalid message, username without asterisk separator": {
			Tx: &weavetest.Tx{
				Msg: &ChangeTokenTargetsMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Username: "alice@iov",
					NewTargets: []BlockchainAddress{
						{BlockchainID: "hydracoin", Address: "some-hydra-address"},
					},
				},
			},
			Auth:           &weavetest.Auth{Signer: aliceCond},
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"invalid message, username name too short": {
			Tx: &weavetest.Tx{
				Msg: &ChangeTokenTargetsMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Username: "a*iov",
					NewTargets: []BlockchainAddress{
						{BlockchainID: "hydracoin", Address: "some-hydra-address"},
					},
				},
			},
			Auth:           &weavetest.Auth{Signer: aliceCond},
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"invalid message, username label too short": {
			Tx: &weavetest.Tx{
				Msg: &ChangeTokenTargetsMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Username: "alice*x",
					NewTargets: []BlockchainAddress{
						{BlockchainID: "hydracoin", Address: "some-hydra-address"},
					},
				},
			},
			Auth:           &weavetest.Auth{Signer: aliceCond},
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"invalid message, username name with invalid characters": {
			Tx: &weavetest.Tx{
				Msg: &ChangeTokenTargetsMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Username: "ALICE*iov",
					NewTargets: []BlockchainAddress{
						{BlockchainID: "hydracoin", Address: "some-hydra-address"},
					},
				},
			},
			Auth:           &weavetest.Auth{Signer: aliceCond},
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"invalid message, username label with invalid characters": {
			Tx: &weavetest.Tx{
				Msg: &ChangeTokenTargetsMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Username: "alice*IOV",
					NewTargets: []BlockchainAddress{
						{BlockchainID: "hydracoin", Address: "some-hydra-address"},
					},
				},
			},
			Auth:           &weavetest.Auth{Signer: aliceCond},
			WantCheckErr:   errors.ErrInput,
			WantDeliverErr: errors.ErrInput,
		},
		"only the owner can change the token": {
			Tx: &weavetest.Tx{
				Msg: &ChangeTokenTargetsMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Username: "alice*iov",
					NewTargets: []BlockchainAddress{
						{BlockchainID: "hydracoin", Address: "some-hydra-address"},
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
					Metadata: &weave.Metadata{Schema: 1},
					Username: "does-not-exist*iov",
					NewTargets: []BlockchainAddress{
						{BlockchainID: "hydracoin", Address: "some-hydra-address"},
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

			config := Configuration{
				ValidUsernameName:  `[a-z0-9\-_.]{3,64}`,
				ValidUsernameLabel: `[a-z0-9]{3,16}`,
			}
			if err := gconf.Save(db, "username", &config); err != nil {
				t.Fatalf("cannot save configuration: %s", err)
			}

			tokens := NewTokenBucket()
			_, err := tokens.Put(db, []byte("alice*iov"), &Token{
				Metadata: &weave.Metadata{Schema: 1},
				Targets: []BlockchainAddress{
					{BlockchainID: "unichain", Address: "some-unicorn-address"},
				},
				Owner: aliceCond.Address(),
			})
			assert.Nil(t, err)

			h := changeTokenTargetsHandler{
				auth:   tc.Auth,
				tokens: tokens,
			}

			cache := db.CacheWrap()
			if _, err := h.Check(context.TODO(), cache, tc.Tx); !tc.WantCheckErr.Is(err) {
				t.Fatalf("unexpected check error: %s", err)
			}
			cache.Discard()
			if _, err := h.Deliver(context.TODO(), db, tc.Tx); !tc.WantDeliverErr.Is(err) {
				t.Fatalf("unexpected deliver error: %s", err)
			}
		})
	}
}
