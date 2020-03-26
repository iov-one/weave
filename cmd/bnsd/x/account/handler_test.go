package account

import (
	"bytes"
	"context"
	"crypto/sha256"
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
		adminCond   = weavetest.NewCondition()
		aliceCond   = weavetest.NewCondition()
		bobCond     = weavetest.NewCondition()
		charlieCond = weavetest.NewCondition()
		brokerCond  = weavetest.NewCondition()

		now = weave.UnixTime(1572247483)
	)

	cases := map[string]struct {
		Requests  []Request
		AfterTest func(t *testing.T, db weave.KVStore)
	}{
		"domain name must be unique": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
							Broker:       brokerCond.Address(),
						},
					},
					BlockHeight: 1,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
							Broker:       brokerCond.Address(),
						},
					},
					BlockHeight: 2,
					WantErr:     errors.ErrDuplicate,
				},
			},
		},
		"account name must be unique within a domain scope": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
						},
					},
					BlockHeight: 1,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Owner:    bobCond.Address(),
							Domain:   "wunderland",
							Name:     "bob",
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Owner:    bobCond.Address(),
							Domain:   "wunderland",
							Name:     "bob",
						},
					},
					BlockHeight: 102,
					WantErr:     errors.ErrDuplicate,
				},
				{
					Now:        now + 3,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wonderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
						},
					},
					BlockHeight: 3,
					WantErr:     nil,
				},
				{
					Now:        now + 4,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Owner:    bobCond.Address(),
							Domain:   "wonderland",
							Name:     "bob",
						},
					},
					BlockHeight: 104,
					WantErr:     nil,
				},
			},
		},
		"configuration can be updated": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &UpdateConfigurationMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Patch: &Configuration{
								Metadata:               &weave.Metadata{Schema: 1},
								Owner:                  aliceCond.Address(),
								ValidName:              "^name$",
								ValidDomain:            "^domain$",
								ValidBlockchainID:      `^bid$`,
								ValidBlockchainAddress: `^baddr$`,
								DomainRenew:            1,
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
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
				assert.Equal(t, conf.ValidBlockchainID, "^bid$")
				assert.Equal(t, conf.ValidBlockchainAddress, "^baddr$")
			},
		},
		"only configuration owner can register a domain without a superuser": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{charlieCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        charlieCond.Address(),
							HasSuperuser: false,
							AccountRenew: 1000,
						},
					},
					BlockHeight: 100,
					WantErr:     errors.ErrUnauthorized,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        charlieCond.Address(),
							HasSuperuser: false,
							AccountRenew: 1000,
						},
					},
					BlockHeight: 100 + 1,
					WantErr:     nil,
				},
			},
		},
		"account owner can add a certificate to an account": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: false,
							AccountRenew: 1000,
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
							Owner:    bobCond.Address(),
							Domain:   "wunderland",
							Name:     "bob",
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &AddAccountCertificateMsg{
							Metadata:    &weave.Metadata{Schema: 1},
							Domain:      "wunderland",
							Name:        "bob",
							Certificate: []byte("a certificate"),
						},
					},
					BlockHeight: 102,
					WantErr:     nil,
				},
			},
		},
		"domain admin cannot add a certificate to an account in that domain if not an account owner": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
							Owner:    bobCond.Address(),
							Domain:   "wunderland",
							Name:     "bob",
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &AddAccountCertificateMsg{
							Metadata:    &weave.Metadata{Schema: 1},
							Domain:      "wunderland",
							Name:        "bob",
							Certificate: []byte("a certificate"),
						},
					},
					BlockHeight: 102,
					WantErr:     errors.ErrUnauthorized,
				},
			},
		},
		"changing a domain admin removes all certificates from accounts that belong to that domain": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &AddAccountCertificateMsg{
							Metadata:    &weave.Metadata{Schema: 1},
							Domain:      "wunderland",
							Name:        "",
							Certificate: []byte("first cert"),
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Owner:    bobCond.Address(),
							Domain:   "wunderland",
							Name:     "bob",
						},
					},
					BlockHeight: 102,
					WantErr:     nil,
				},
				{
					Now:        now + 3,
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &AddAccountCertificateMsg{
							Metadata:    &weave.Metadata{Schema: 1},
							Domain:      "wunderland",
							Name:        "bob",
							Certificate: []byte("second cert"),
						},
					},
					BlockHeight: 103,
					WantErr:     nil,
				},

				{
					Now:        now + 4,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &TransferDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							NewAdmin: charlieCond.Address(),
						},
					},
					BlockHeight: 104,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				accounts := NewAccountBucket()
				var empty Account
				if err := accounts.One(db, accountKey("", "wunderland"), &empty); err != nil {
					t.Fatalf("cannot get wunderland account: %s", err)
				}
				if len(empty.Certificates) != 0 {
					t.Errorf("empty account certificates were not removed: %q", empty.Certificates)
				}

				var bob Account
				if err := accounts.One(db, accountKey("bob", "wunderland"), &bob); err != nil {
					t.Fatalf("cannot get wunderland account: %s", err)
				}
				if len(bob.Certificates) != 0 {
					t.Errorf("bob account certificates were not removed: %q", bob.Certificates)
				}
			},
		},
		"changing account owner removes all certificates from that account": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
							Owner:    bobCond.Address(),
							Domain:   "wunderland",
							Name:     "bob",
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &AddAccountCertificateMsg{
							Metadata:    &weave.Metadata{Schema: 1},
							Domain:      "wunderland",
							Name:        "bob",
							Certificate: []byte("a cert"),
						},
					},
					BlockHeight: 102,
					WantErr:     nil,
				},
				{
					Now:        now + 3,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &TransferAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "bob",
							NewOwner: charlieCond.Address(),
						},
					},
					BlockHeight: 103,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				accounts := NewAccountBucket()
				var a Account
				if err := accounts.One(db, accountKey("bob", "wunderland"), &a); err != nil {
					t.Fatalf("cannot get wunderland account: %s", err)
				}
				if len(a.Certificates) != 0 {
					t.Fatalf("certificates were not removed: %q", a.Certificates)
				}
			},
		},
		"certificate of an account that belongs to an expired domain cannot be added": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 900,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Owner:    bobCond.Address(),
							Domain:   "wunderland",
							Name:     "bob",
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now:        now + 1100, // Domain is expired, account not.
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &AddAccountCertificateMsg{
							Metadata:    &weave.Metadata{Schema: 1},
							Domain:      "wunderland",
							Name:        "bob",
							Certificate: []byte("a cert"),
						},
					},
					BlockHeight: 102,
					WantErr:     errors.ErrExpired,
				},
			},
		},
		"certificate of an expired account cannot be added": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 10,
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
							Owner:    bobCond.Address(),
							Domain:   "wunderland",
							Name:     "bob",
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now:        now + 200, // Domain is not expired, but account is.
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &AddAccountCertificateMsg{
							Metadata:    &weave.Metadata{Schema: 1},
							Domain:      "wunderland",
							Name:        "bob",
							Certificate: []byte("a cert"),
						},
					},
					BlockHeight: 102,
					WantErr:     errors.ErrExpired,
				},
			},
		},
		"changing account targets does not removes certificates from that account": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
							Owner:    bobCond.Address(),
							Domain:   "wunderland",
							Name:     "bob",
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &AddAccountCertificateMsg{
							Metadata:    &weave.Metadata{Schema: 1},
							Domain:      "wunderland",
							Name:        "bob",
							Certificate: []byte("a cert"),
						},
					},
					BlockHeight: 102,
					WantErr:     nil,
				},
				{
					Now:        now + 3,
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &ReplaceAccountTargetsMsg{
							Metadata:   &weave.Metadata{Schema: 1},
							Domain:     "wunderland",
							Name:       "bob",
							NewTargets: nil,
						},
					},
					BlockHeight: 103,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				accounts := NewAccountBucket()
				var a Account
				if err := accounts.One(db, accountKey("bob", "wunderland"), &a); err != nil {
					t.Fatalf("cannot get wunderland account: %s", err)
				}
				if len(a.Certificates) != 1 {
					t.Fatal("certificates were removed")
				}
			},
		},
		"anyone can register a domain with superuser": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        bobCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
				if !d.Admin.Equals(bobCond.Address()) {
					t.Fatalf("unexpected wunderland owner: want %q, got %q", bobCond.Address(), d.Admin)
				}
				if got, want := d.ValidUntil, weave.UnixTime(now+1000); got != want {
					t.Fatalf("unexpected valid till: want %d, got %d", want, got)
				}
			},
		},
		"registering a domain creates an account with an empty name": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
					t.Fatalf("cannot get wunderland account: %s", err)
				}
				if a.Name != "" {
					t.Fatalf("want an empty name, got %q", a.Name)
				}
				if a.Domain != "wunderland" {
					t.Fatalf("want wunderland domain, got %q", a.Domain)
				}
				if !a.Owner.Equals(aliceCond.Address()) {
					t.Fatalf("want alice to be an owner, got %q", a.Owner)
				}
			},
		},
		"deletion of the empty name account is not possible": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
		"an account owner can delete account": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
							Owner:    bobCond.Address(),
							Domain:   "wunderland",
							Name:     "bob",
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{charlieCond},
					Tx: &weavetest.Tx{
						Msg: &DeleteAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "bob",
						},
					},
					BlockHeight: 102,
					WantErr:     errors.ErrUnauthorized,
				},
				{
					Now:        now + 3,
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
		"a domain admin can delete any account in that domain when the domain has superuser": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{aliceCond, bobCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Owner:    bobCond.Address(),
							Domain:   "wunderland",
							Name:     "bob",
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &DeleteAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "bob",
						},
					},
					BlockHeight: 102,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				assertAccounts(t, db, "wunderland", []string{"*wunderland"})
			},
		},
		"a domain admin cannot delete an account in that domain when the domain does not have superuser": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: false,
							AccountRenew: 1000,
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{aliceCond, bobCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Owner:    bobCond.Address(),
							Domain:   "wunderland",
							Name:     "bob",
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &DeleteAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "bob",
						},
					},
					BlockHeight: 102,
					WantErr:     errors.ErrUnauthorized,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				assertAccounts(t, db, "wunderland", []string{"*wunderland", "bob*wunderland"})
			},
		},
		"only a domain admin can register account under a domain that has superuser": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
		"anyone can register an account under a domain without a superuser": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: false,
							AccountRenew: 1000,
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
							Owner:    bobCond.Address(),
							Name:     "bob",
						},
					},
					BlockHeight: 101,
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
							Owner:    charlieCond.Address(),
						},
					},
					BlockHeight: 102,
					WantErr:     errors.ErrUnauthorized,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				assertAccounts(t, db, "wunderland", []string{"*wunderland", "bob*wunderland"})
			},
		},
		"deletion of a domain deletes all accounts that domain contained": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
		"domain without a superuser cannot be deleted": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        adminCond.Address(),
							HasSuperuser: false,
							AccountRenew: 1000,
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &DeleteDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 102,
					WantErr:     errors.ErrState,
				},
			},
		},
		"deleting all accounts (flushing) does not delete the empty name account": {
			Requests: []Request{
				{
					Now:        now + 1,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Admin:        aliceCond.Address(),
							Domain:       "wunderland",
							HasSuperuser: true,
							AccountRenew: 1000,
						},
					},
					BlockHeight: 101,
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
						Msg: &RegisterAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "bob",
							Owner:    bobCond.Address(),
						},
					},
					BlockHeight: 103,
					WantErr:     nil,
				},
				{
					Now:        now + 4,
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &FlushDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 104,
					WantErr:     errors.ErrUnauthorized, // Only the domain admin can delete.
				},
				{
					Now:        now + 5,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &FlushDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
						},
					},
					BlockHeight: 105,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				// Deleting all accounts does not delete the one with an empty name.
				assertAccounts(t, db, "wunderland", []string{"*wunderland"})
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
		"iov domain can be registered": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "iov",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
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
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
				if got, want := d.ValidUntil, weave.UnixTime(now+2+1000); want != got {
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
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &UpdateConfigurationMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Patch: &Configuration{
								Metadata:               &weave.Metadata{Schema: 1},
								Owner:                  aliceCond.Address(),
								ValidName:              `^[a-z]+$`,
								ValidDomain:            `^[a-z]+$`,
								ValidBlockchainID:      `^[a-z]+$`,
								ValidBlockchainAddress: `^[a-z]+$`,
								DomainRenew:            1,
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
				if got, want := d.ValidUntil, weave.UnixTime(now+1000); want != got {
					t.Logf("want %d %s", want, want)
					t.Logf(" got %d %s", got, got)
					t.Fatal("unexpected valid till")
				}
			},
		},
		"anyone can renew an account": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							AccountRenew: 21,
							HasSuperuser: true,
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
							Owner:    bobCond.Address(),
							Domain:   "wunderland",
							Name:     "bob",
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{},
					Tx: &weavetest.Tx{
						Msg: &RenewAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Name:     "bob",
							Domain:   "wunderland",
						},
					},
					BlockHeight: 102,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				b := NewAccountBucket()
				var a Account
				if err := b.One(db, accountKey("bob", "wunderland"), &a); err != nil {
					t.Fatalf("cannot get wunderland account: %s", err)
				}
				// Expiration time should be execution time
				// (block time) which is now + 2, plus
				// expiration offset.
				if got, want := a.ValidUntil, weave.UnixTime(now+2+21); want != got {
					t.Fatalf("want valid till %s, got %s", want, got)
				}
			},
		},
		"a domain can be deleted by the domain admin that has a superuser": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
		"domain without a superuser cannot be transferred": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        adminCond.Address(),
							HasSuperuser: false,
							AccountRenew: 1000,
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &TransferDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							NewAdmin: bobCond.Address(),
						},
					},
					BlockHeight: 102,
					WantErr:     errors.ErrState,
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
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
		"a domain ownership (admin address) can be transferred by the domain admin": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
							NewAdmin: bobCond.Address(),
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
				if !d.Admin.Equals(bobCond.Address()) {
					t.Fatalf("unexpected owner: %q", d.Admin)
				}
			},
		},
		"accounts ownership is transferred to new domain owner after clearing certs and targets": {
			Requests: []Request{
				// register domain
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				// add account to domain
				{
					Now:        now + 1,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "test-account",
							Owner:    aliceCond.Address(),
							Targets: []BlockchainAddress{
								{
									BlockchainID: "blockchain-id",
									Address:      "blockchain-address",
								},
							},
							Broker: nil,
						},
						Err: nil,
					},
					BlockHeight: 101,
				},
				// add certs to to account
				{
					Now:        now + 2,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &AddAccountCertificateMsg{
							Metadata:    &weave.Metadata{Schema: 1},
							Domain:      "wunderland",
							Name:        "test-account",
							Certificate: []byte("a-mock-certificate"),
						},
						Err: nil,
					},
					BlockHeight: 102,
					WantErr:     nil,
				},
				// transfer domain
				{
					Now:        now + 3,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &TransferDomainMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							NewAdmin: bobCond.Address(),
						},
					},
					BlockHeight: 103,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				domainBucket := NewDomainBucket()
				accountsBucket := NewAccountBucket()
				var d Domain
				if err := domainBucket.One(db, []byte("wunderland"), &d); err != nil {
					t.Fatalf("cannot get wunderland domain: %s", err)
				}
				// create iterator
				iterator := domainAccountIter{
					db:       db,
					domain:   []byte("wunderland"),
					accounts: accountsBucket,
				}
				// check if account ownership was correctly transferred
				for {
					switch acc, err := iterator.Next(); {
					case err == nil:
						// check if an account has had an ownership change
						if !bobCond.Address().Equals(acc.Owner) {
							t.Fatalf("account ownership not changed for account %#v, expected: %s, got: %s", acc, bobCond.Address(), acc.Owner)
						}
						// check if certs were cleared
						if len(acc.Certificates) != 0 {
							t.Fatalf("account certificates were not cleared")
						}
						if len(acc.Targets) != 0 {
							t.Fatalf("account targets were not cleared")
						}
					// case we finish iterating
					case errors.ErrIteratorDone.Is(err):
						return
					default:
						t.Fatalf("iterator error: %s", err)
					}
				}
			},
		},
		"expired domain ownership (domain admin) cannot be changed": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
							NewAdmin: bobCond.Address(),
						},
					},
					BlockHeight: 101,
					WantErr:     errors.ErrExpired,
				},
			},
		},
		"a certificate (identified by its content) must not appear more than once in the list": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &AddAccountCertificateMsg{
							Metadata:    &weave.Metadata{Schema: 1},
							Domain:      "wunderland",
							Name:        "",
							Certificate: []byte("a certificate"),
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &AddAccountCertificateMsg{
							Metadata:    &weave.Metadata{Schema: 1},
							Domain:      "wunderland",
							Name:        "",
							Certificate: []byte("a certificate"),
						},
					},
					BlockHeight: 102,
					WantErr:     errors.ErrDuplicate,
				},
				{
					Now:        now + 3,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &AddAccountCertificateMsg{
							Metadata:    &weave.Metadata{Schema: 1},
							Domain:      "wunderland",
							Name:        "",
							Certificate: []byte("another certificate"),
						},
					},
					BlockHeight: 103,
					WantErr:     nil,
				},
			},
		},
		"a domain admin can change account targets only when no account owner is set": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
					Conditions: []weave.Condition{aliceCond},
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
					WantErr:     errors.ErrUnauthorized,
				},
				{
					Now:        now + 3,
					Conditions: []weave.Condition{bobCond},
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
					BlockHeight: 103,
					WantErr:     nil,
				},
				{
					Now:        now + 4,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &TransferAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "bob",
							NewOwner: aliceCond.Address(),
						},
					},
					BlockHeight: 104,
					WantErr:     nil,
				},
				{
					Now:        now + 5,
					Conditions: []weave.Condition{aliceCond}, // Signed by the domain admin, NOT by the account owner.
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
					BlockHeight: 105,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				accounts := NewAccountBucket()
				var a Account
				if err := accounts.One(db, accountKey("bob", "wunderland"), &a); err != nil {
					t.Fatalf("cannot get wunderland account: %s", err)
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
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
					Conditions: []weave.Condition{bobCond}, // Signed by the account owner (not domain admin).
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
					t.Fatalf("cannot get wunderland account: %s", err)
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
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
					t.Fatalf("cannot get wunderland account: %s", err)
				}
				if len(a.Targets) != 0 {
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
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
					Now:        now + 1000000000, // Expired.
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
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
					Now:        now + 100000000, // Expired.
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &TransferAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "bob",
							NewOwner: charlieCond.Address(),
						},
					},
					BlockHeight: 124,
					WantErr:     errors.ErrExpired,
				},
			},
		},
		"account that belongs to an expired domain cannot be transferred": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 100000,
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1000 - 2, // Close to domain expiration.
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
					Now:        now + 1000 + 5, // Domain is expired, not the account.
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &TransferAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "bob",
							NewOwner: charlieCond.Address(),
						},
					},
					BlockHeight: 124,
					WantErr:     errors.ErrExpired,
				},
			},
		},
		"only account owner can transfer an account that belongs to a domain with no superuser": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: false,
							AccountRenew: 1000,
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
				// Domain admin cannot transfer an account.
				{
					Now:        now + 3,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &TransferAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "bob",
							NewOwner: bobCond.Address(),
						},
					},
					BlockHeight: 103,
					WantErr:     errors.ErrUnauthorized,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				accounts := NewAccountBucket()
				var a Account
				if err := accounts.One(db, accountKey("bob", "wunderland"), &a); err != nil {
					t.Fatalf("cannot get wunderland account: %s", err)
				}
				if !a.Owner.Equals(charlieCond.Address()) {
					t.Fatalf("want the owner to be %q, got %q", charlieCond.Address(), a.Owner)
				}
			},
		},
		"account owner cannot transfer ownership of an account that belong to a domain with a superuser": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
				// Account owner cannot transfer.
				{
					Now:        now + 2,
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &TransferAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "bob",
							NewOwner: aliceCond.Address(),
						},
					},
					BlockHeight: 102,
					WantErr:     errors.ErrUnauthorized,
				},
				// Domain admin can transfer.
				{
					Now:        now + 3,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &TransferAccountMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Domain:   "wunderland",
							Name:     "bob",
							NewOwner: charlieCond.Address(),
						},
					},
					BlockHeight: 103,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				accounts := NewAccountBucket()
				var a Account
				if err := accounts.One(db, accountKey("bob", "wunderland"), &a); err != nil {
					t.Fatalf("cannot get wunderland account: %s", err)
				}
				if !a.Owner.Equals(charlieCond.Address()) {
					t.Fatalf("want the owner to be %q, got %q", charlieCond.Address(), a.Owner)
				}
			},
		},
		"an account owner can delete a single certificate identified by sha256 hash of the content": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &RegisterDomainMsg{
							Metadata:     &weave.Metadata{Schema: 1},
							Domain:       "wunderland",
							Admin:        aliceCond.Address(),
							HasSuperuser: true,
							AccountRenew: 1000,
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
							Owner:    bobCond.Address(),
							Domain:   "wunderland",
							Name:     "bob",
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &AddAccountCertificateMsg{
							Metadata:    &weave.Metadata{Schema: 1},
							Domain:      "wunderland",
							Name:        "bob",
							Certificate: []byte("first certificate"),
						},
					},
					BlockHeight: 102,
					WantErr:     nil,
				},
				{
					Now:        now + 3,
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &AddAccountCertificateMsg{
							Metadata:    &weave.Metadata{Schema: 1},
							Domain:      "wunderland",
							Name:        "bob",
							Certificate: []byte("second certificate"),
						},
					},
					BlockHeight: 103,
					WantErr:     nil,
				},
				{
					Now:        now + 4,
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &DeleteAccountCertificateMsg{
							Metadata:        &weave.Metadata{Schema: 1},
							Domain:          "wunderland",
							Name:            "bob",
							CertificateHash: checksum256(t, "first certificate"),
						},
					},
					BlockHeight: 104,
					WantErr:     nil,
				},
				{
					Now:        now + 5,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &DeleteAccountCertificateMsg{
							Metadata:        &weave.Metadata{Schema: 1},
							Domain:          "wunderland",
							Name:            "bob",
							CertificateHash: checksum256(t, "second certificate"),
						},
					},
					BlockHeight: 105,
					WantErr:     errors.ErrUnauthorized,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				accounts := NewAccountBucket()
				var a Account
				if err := accounts.One(db, accountKey("bob", "wunderland"), &a); err != nil {
					t.Fatalf("cannot get wunderland account: %s", err)
				}
				if len(a.Certificates) != 1 {
					t.Fatalf("want one certificate, got %q", a.Certificates)
				}
				if !bytes.Equal(a.Certificates[0], []byte("second certificate")) {
					t.Errorf("unexpected certificate: %q", a.Certificates)
				}
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, "account")

			rt := app.NewRouter()
			auth := &weavetest.CtxAuth{Key: "auth"}
			RegisterRoutes(rt, auth)

			config := Configuration{
				Metadata:               &weave.Metadata{Schema: 1},
				Owner:                  adminCond.Address(),
				ValidName:              `^[a-z0-9\-_.]{0,64}$`,
				ValidDomain:            `^[a-z0-9]{3,16}$`,
				ValidBlockchainID:      `^[a-z0-9]{2,64}$`,
				ValidBlockchainAddress: `^[a-z0-9]{3,128}$`,
				DomainRenew:            1000,
			}
			if err := gconf.Save(db, "account", &config); err != nil {
				t.Fatalf("cannot save configuration: %s", err)
			}

			for i, req := range tc.Requests {
				ctx := weave.WithHeight(context.Background(), req.BlockHeight)
				ctx = weave.WithChainID(ctx, "testchain-123")
				ctx = auth.SetConditions(ctx, req.Conditions...)
				ctx = weave.WithBlockTime(ctx, req.Now.Time())

				cache := db.CacheWrap()
				if _, err := rt.Check(ctx, cache, req.Tx); !req.WantErr.Is(err) {
					t.Fatalf("unexpected %d check error: want %q, got %+v", i, req.WantErr, err)
				}
				cache.Discard()
				if _, err := rt.Deliver(ctx, db, req.Tx); !req.WantErr.Is(err) {
					t.Fatalf("unexpected %d deliver error: want %q, got %+v", i, req.WantErr, err)
				}
			}

			if tc.AfterTest != nil {
				tc.AfterTest(t, db)
			}
		})
	}
}

func checksum256(t testing.TB, s string) []byte {
	t.Helper()
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}

func assertAccounts(t testing.TB, db weave.ReadOnlyKVStore, domain string, wantAccounts []string) {
	t.Helper()

	var accs []*Account
	_, err := NewAccountBucket().ByIndex(db, "domain", []byte(domain), &accs)
	if err != nil {
		t.Fatalf("cannot list accounts for domain %q: %s", domain, err)
	}

	var accounts []string
	for _, a := range accs {
		accounts = append(accounts, a.Name+"*"+a.Domain)
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
