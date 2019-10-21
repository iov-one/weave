package blueaccount

import (
	"context"
	"reflect"
	"sort"
	"testing"

	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestUseCases(t *testing.T) {
	type Request struct {
		Now         weave.UnixTime
		Conditions  []weave.Condition
		Tx          weave.Tx
		BlockHeight int64
		WantErr     *errors.Error
	}

	var (
		aliceCond   = weavetest.NewCondition()
		bobCond     = weavetest.NewCondition()
		charlieCond = weavetest.NewCondition()

		now = weave.UnixTime(1572247483)
	)

	cases := map[string]struct {
		Requests  []Request
		AfterTest func(t *testing.T, db weave.KVStore)
	}{
		"configuration can be updated": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &UpdateConfigurationMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Patch: &Configuration{
								Metadata:    &weave.Metadata{Schema: 1},
								Owner:       aliceCond.Address(),
								ValidName:   "^name$",
								ValidDomain: "^domain$",
								DomainRenew: 1,
							},
						},
					},
					BlockHeight: 3,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 4,
					WantErr:     errors.ErrInput,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				conf, err := loadConf(db)
				if err != nil {
					t.Fatalf("cannot load configuration: %s", err)
				}
				assert.Equal(t, conf.ValidName, "^name$")
				assert.Equal(t, conf.ValidDomain, "^domain$")
				assert.Equal(t, conf.DomainRenew, weave.UnixDuration(1))
			},
		},
		"anyone can register a domain, main signer will be the owner": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				b := NewDomainBucket()
				var d Domain
				if err := b.One(db, []byte("wunderland"), &d); err != nil {
					t.Fatalf("cannot get wunderland domain: %s", err)
				}
				if d.Domain != "wunderland" {
					t.Fatalf("unexpected wunderland domain: want wunderland, got %q", d.Domain)
				}
				if !d.Owner.Equals(aliceCond.Address()) {
					t.Fatalf("unexpected wunderland owner: want %q, got %q", aliceCond.Address(), d.Owner)
				}
				if got, want := d.ValidTill, weave.UnixTime(now+1000); got != want {
					t.Fatalf("unexpected valid till: want %d, got %d", want, got)
				}
			},
		},
		"anyone can register a domain, main signer may delegate the ownership": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Owner:    bobCond.Address(),
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				b := NewDomainBucket()
				var d Domain
				if err := b.One(db, []byte("wunderland"), &d); err != nil {
					t.Fatalf("cannot get wunderland domain: %s", err)
				}
				if d.Domain != "wunderland" {
					t.Fatalf("unexpected wunderland domain: want wunderland, got %q", d.Domain)
				}
				if !d.Owner.Equals(bobCond.Address()) {
					t.Fatalf("unexpected wunderland owner: want %q, got %q", bobCond.Address(), d.Owner)
				}
				if got, want := d.ValidTill, weave.UnixTime(now+1000); got != want {
					t.Fatalf("unexpected valid till: want %d, got %d", want, got)
				}
			},
		},
		"registering a domain creates a username with an empty name": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				accounts := NewAccountBucket()
				var a Account
				if err := accounts.One(db, accountKey("", "wunderland"), &a); err != nil {
					t.Fatalf("cannot get wunderland username: %s", err)
				}
				if a.Name != "" {
					t.Fatalf("want an empty name, got %q", a.Name)
				}
				if a.Domain != "wunderland" {
					t.Fatalf("want wunderland domain, got %q", a.Domain)
				}
				if a.Owner != nil {
					t.Fatalf("want nil owner, got %q", a.Owner)
				}
			},
		},
		"deletion of the empty name username is not possible": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &DeleteAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "",
						},
					},
					BlockHeight: 101,
					WantErr:     errors.ErrState,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				assertAccounts(t, db, "wunderland", []string{"*wunderland"})
			},
		},
		"only owner can register username under a domain": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "bob",
						},
					},
					BlockHeight: 101,
					WantErr:     errors.ErrUnauthorized,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "alice",
						},
					},
					BlockHeight: 102,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				assertAccounts(t, db, "wunderland", []string{"*wunderland", "alice*wunderland"})
			},
		},
		"deletion of a domain deletes all username that domain contains": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "alice",
						},
					},
					BlockHeight: 102,
					WantErr:     nil,
				},
				{
					Now:        now + 3,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &DeleteDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 103,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				assertAccounts(t, db, "wunderland", nil)
			},
		},
		"deletion of a non existing domain fails": {
			Requests: []Request{
				{
					Now:        now + 4,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &DeleteDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 104,
					WantErr:     errors.ErrNotFound,
				},
			},
		},
		"iov domain cannot be registered": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "iov",
						},
					},
					BlockHeight: 100,
					WantErr:     errors.ErrInput,
				},
			},
		},
		"anyone can renew a domain": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RenewDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 102,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				b := NewDomainBucket()
				var d Domain
				if err := b.One(db, []byte("wunderland"), &d); err != nil {
					t.Fatalf("cannot get wunderland domain: %s", err)
				}
				// Expiration time should be execution time
				// (block time) which is now + 2, plus
				// expiration offset.
				if got, want := d.ValidTill, weave.UnixTime(now+2+1000); want != got {
					t.Fatalf("want valid till %s, got %s", want, got)
				}
			},
		},
		"renewing a domain does not shorten its expiration time": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &UpdateConfigurationMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Patch: &Configuration{
								Metadata:    &weave.Metadata{Schema: 1},
								Owner:       aliceCond.Address(),
								DomainRenew: 1,
							},
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RenewDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 102,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				b := NewDomainBucket()
				var d Domain
				if err := b.One(db, []byte("wunderland"), &d); err != nil {
					t.Fatalf("cannot get wunderland domain: %s", err)
				}
				// Expiration time should not be updated
				// because it would be shortened.
				if got, want := d.ValidTill, weave.UnixTime(now+1000); want != got {
					t.Logf("want %d %s", want, want)
					t.Logf(" got %d %s", got, got)
					t.Fatal("unexpected valid till")
				}
			},
		},
		"a domain can be deleted by the owner": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &DeleteDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
			},
		},
		"a new account cannot be registered under an expired domain": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 10002, // Domain is expired by now.
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "alice",
						},
					},
					BlockHeight: 102,
					WantErr:     errors.ErrExpired,
				},
			},
		},
		"domain ownership can be transferred": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &TransferDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							NewOwner: bobCond.Address(),
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				b := NewDomainBucket()
				var d Domain
				if err := b.One(db, []byte("wunderland"), &d); err != nil {
					t.Fatalf("cannot get wunderland domain: %s", err)
				}
				if !d.Owner.Equals(bobCond.Address()) {
					t.Fatalf("unexpected owner: %q", d.Owner)
				}
			},
		},
		"expired domain ownership canot be transferred": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1000000000, // Domain is expired.
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &TransferDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							NewOwner: bobCond.Address(),
						},
					},
					BlockHeight: 101,
					WantErr:     errors.ErrExpired,
				},
			},
		},
		"a domain owner can change any account targets": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "bob",
							Owner:    bobCond.Address(),
							Targets: []BlockchainAddress{
								{BlockchainID: "unicoin", Address: "abc123"},
							},
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{aliceCond}, // Signed by the domain owner, NOT by the account owner.
					Tx: &weavetest.Tx{
						Msg: &ReplaceAccountTargetsMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "bob",
							NewTargets: []BlockchainAddress{
								{BlockchainID: "doge", Address: "987xyz"},
							},
						},
					},
					BlockHeight: 102,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				accounts := NewAccountBucket()
				var a Account
				if err := accounts.One(db, accountKey("bob", "wunderland"), &a); err != nil {
					t.Fatalf("cannot get wunderland username: %s", err)
				}
				assert.Equal(t, a.Targets, []BlockchainAddress{
					{BlockchainID: "doge", Address: "987xyz"},
				})
			},
		},
		"an account owner can change the list of targets": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "bob",
							Owner:    bobCond.Address(),
							Targets: []BlockchainAddress{
								{BlockchainID: "unicoin", Address: "abc123"},
							},
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{bobCond}, // Signed by the account owner (not domain owner).
					Tx: &weavetest.Tx{
						Msg: &ReplaceAccountTargetsMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "bob",
							NewTargets: []BlockchainAddress{
								{BlockchainID: "doge", Address: "987xyz"},
							},
						},
					},
					BlockHeight: 102,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				accounts := NewAccountBucket()
				var a Account
				if err := accounts.One(db, accountKey("bob", "wunderland"), &a); err != nil {
					t.Fatalf("cannot get wunderland username: %s", err)
				}
				assert.Equal(t, a.Targets, []BlockchainAddress{
					{BlockchainID: "doge", Address: "987xyz"},
				})
			},
		},
		"expired account target cannot be changed": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1000000, // Long after the expiration period.
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &ReplaceAccountTargetsMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "",
							NewTargets: []BlockchainAddress{
								{BlockchainID: "doge", Address: "987xyz"},
							},
						},
					},
					BlockHeight: 102,
					WantErr:     errors.ErrExpired,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				accounts := NewAccountBucket()
				var a Account
				if err := accounts.One(db, accountKey("", "wunderland"), &a); err != nil {
					t.Fatalf("cannot get wunderland username: %s", err)
				}
				if len(a.Targets) != 0 {
					t.Fatalf("expected no targets, got %+v", a.Targets)
				}
			},
		},
		"expired account can be deleted": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "bob",
							Owner:    bobCond.Address(),
							Targets: []BlockchainAddress{
								{BlockchainID: "unicoin", Address: "abc123"},
							},
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now:        now + 1000000000, // Domain is expired.
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &DeleteAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "bob",
						},
					},
					BlockHeight: 103,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				assertAccounts(t, db, "wunderland", []string{"*wunderland"})
			},
		},
		"empty name account cannot be transferred": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &TransferAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "",
							NewOwner: bobCond.Address(),
						},
					},
					BlockHeight: 101,
					WantErr:     errors.ErrInput,
				},
			},
		},
		"expired account cannot be transferred": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 100000000, // Domain is expired.
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "bob",
							Owner:    bobCond.Address(),
						},
					},
					BlockHeight: 101,
					WantErr:     errors.ErrExpired,
				},
			},
		},
		"owner can transfer ownership of an account": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "bob",
							Owner:    bobCond.Address(),
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &TransferAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "bob",
							NewOwner: charlieCond.Address(),
						},
					},
					BlockHeight: 102,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				accounts := NewAccountBucket()
				var a Account
				if err := accounts.One(db, accountKey("bob", "wunderland"), &a); err != nil {
					t.Fatalf("cannot get wunderland username: %s", err)
				}
				if !a.Owner.Equals(charlieCond.Address()) {
					t.Fatalf("want the owner to be %q, got %q", charlieCond.Address(), a.Owner)
				}
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, "blueaccount")

			rt := app.NewRouter()
			auth := &weavetest.CtxAuth{Key: "auth"}
			RegisterRoutes(rt, auth)

			config := Configuration{
				Metadata:    &weave.Metadata{Schema: 1},
				Owner:       aliceCond.Address(),
				ValidName:   `^[a-z0-9\-_.]{0,64}$`,
				ValidDomain: `^[a-z0-9]{3,16}$`,
				DomainRenew: 1000,
			}
			if err := gconf.Save(db, "blueaccount", &config); err != nil {
				t.Fatalf("cannot save configuration: %s", err)
			}

			for _, req := range tc.Requests {
				ctx := weave.WithHeight(context.Background(), req.BlockHeight)
				ctx = weave.WithChainID(ctx, "testchain-123")
				ctx = auth.SetConditions(ctx, req.Conditions...)
				ctx = weave.WithBlockTime(ctx, req.Now.Time())

				cache := db.CacheWrap()
				if _, err := rt.Check(ctx, cache, req.Tx); !req.WantErr.Is(err) {
					t.Fatalf("unexpected check error: want %q, got %+v", req.WantErr, err)
				}
				cache.Discard()
				if _, err := rt.Deliver(ctx, db, req.Tx); !req.WantErr.Is(err) {
					t.Fatalf("unexpected deliver error: want %q, got %+v", req.WantErr, err)
				}
			}

			if tc.AfterTest != nil {
				tc.AfterTest(t, db)
			}
		})
	}
}

func TestDomainAccounts(t *testing.T) {
	db := store.MemStore()
	migration.MustInitPkg(db, "blueaccount")

	// Two domains 'a' and 'ab' are similar to ensure that domain/name
	// separation works correctly. If it does not, 'a' + 'bbb' should
	// produce the same account as 'ab' + 'bb'.

	domains := NewDomainBucket()
	accounts := NewAccountBucket()

	domains.Put(db, []byte("a"), &Domain{
		Metadata: &weave.Metadata{},
		Owner:    weavetest.NewCondition().Address(),
		Domain:   "a",
	})
	accounts.Put(db, accountKey("", "a"), &Account{
		Metadata: &weave.Metadata{},
		Owner:    weavetest.NewCondition().Address(),
		Domain:   "a",
		Name:     "",
	})
	accounts.Put(db, accountKey("bbb", "a"), &Account{
		Metadata: &weave.Metadata{},
		Owner:    weavetest.NewCondition().Address(),
		Domain:   "a",
		Name:     "bbb",
	})

	domains.Put(db, []byte("ab"), &Domain{
		Metadata: &weave.Metadata{},
		Owner:    weavetest.NewCondition().Address(),
		Domain:   "ab",
	})
	accounts.Put(db, accountKey("", "ab"), &Account{
		Metadata: &weave.Metadata{},
		Owner:    weavetest.NewCondition().Address(),
		Domain:   "ab",
		Name:     "",
	})
	accounts.Put(db, accountKey("bb", "ab"), &Account{
		Metadata: &weave.Metadata{},
		Owner:    weavetest.NewCondition().Address(),
		Domain:   "ab",
		Name:     "bb",
	})
	accounts.Put(db, accountKey("xyz", "ab"), &Account{
		Metadata: &weave.Metadata{},
		Owner:    weavetest.NewCondition().Address(),
		Domain:   "ab",
		Name:     "xyz",
	})

	assertAccounts(t, db, "a", []string{"*a", "bbb*a"})
	assertAccounts(t, db, "ab", []string{"*ab", "bb*ab", "xyz*ab"})
}

func assertAccounts(t testing.TB, db weave.ReadOnlyKVStore, domain string, wantAccounts []string) {
	t.Helper()

	iter, err := DomainAccounts(db, domain)
	if err != nil {
		t.Fatalf("cannot list %q domain accounts", domain)
	}
	defer iter.Release()

	var accounts []string

iterAccounts:
	for {
		switch key, raw, err := iter.Next(); {
		case err == nil:
			var a Account
			if err := a.Unmarshal(raw); err != nil {
				t.Fatalf("cannot unmarshal %q account: %s", key, err)
			}
			accounts = append(accounts, a.Name+"*"+a.Domain)
		case errors.ErrIteratorDone.Is(err):
			break iterAccounts
		default:
			t.Fatalf("cannot get next account name: %s", err)
		}
	}

	// Order does not matter. This is only membership test.
	sort.Strings(wantAccounts)
	sort.Strings(accounts)

	if !reflect.DeepEqual(accounts, wantAccounts) {
		t.Logf("want accounts %d: %q", len(wantAccounts), wantAccounts)
		t.Logf(" got accounts %d: %q", len(accounts), accounts)
		t.Fatal("unexpected accounts")
	}
}
