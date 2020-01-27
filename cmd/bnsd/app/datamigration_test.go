package bnsd

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/x/account"
	"github.com/iov-one/weave/cmd/bnsd/x/preregistration"
	"github.com/iov-one/weave/cmd/bnsd/x/username"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestRewriteUsernameAccounts(t *testing.T) {
	db := store.MemStore()
	migration.MustInitPkg(db, "datamigration", "username", "account")

	ctx := context.Background()
	ctx = weave.WithBlockTime(ctx, time.Now())

	var (
		aliceCond = weave.NewCondition("alice", "test", []byte{1})
		bobCond   = weave.NewCondition("bob", "test", []byte{1})
	)

	tokens := username.NewTokenBucket()
	_, err := tokens.Put(db, []byte("alice*iov"), &username.Token{
		Metadata: &weave.Metadata{Schema: 1},
		Owner:    aliceCond.Address(),
		Targets: []username.BlockchainAddress{
			{BlockchainID: "blockchain1", Address: "addr1"},
			{BlockchainID: "blockchain2", Address: "addr2"},
		},
	})
	if err != nil {
		t.Fatalf("cannot store alice: %s", err)
	}
	_, err = tokens.Put(db, []byte("bob*iov"), &username.Token{
		Metadata: &weave.Metadata{Schema: 1},
		Owner:    bobCond.Address(),
		Targets:  nil,
	})
	if err != nil {
		t.Fatalf("cannot store bob: %s", err)
	}

	if err := rewriteUsernameAccounts(ctx, db); err != nil {
		t.Fatalf("cannot rewrite username accounts: %s", err)
	}

	domains := account.NewDomainBucket()
	var d account.Domain
	if err := domains.One(db, []byte("iov"), &d); err != nil {
		t.Fatalf("cannot get iov domain: %s", err)
	}
	assert.Equal(t, d.Domain, "iov")
	assert.Equal(t, d.HasSuperuser, false)

	accounts := account.NewAccountBucket()

	var empty account.Account
	if err := accounts.One(db, []byte("*iov"), &empty); err != nil {
		t.Fatalf("cannot get empty account: %s", err)
	}

	var alice account.Account
	if err := accounts.One(db, []byte("alice*iov"), &alice); err != nil {
		t.Fatalf("cannot get alice account: %s", err)
	}
	assert.Equal(t, alice.Domain, "iov")
	assert.Equal(t, alice.Name, "alice")
	assert.Equal(t, alice.Owner, aliceCond.Address())
	assert.Equal(t, alice.Targets, []account.BlockchainAddress{
		{BlockchainID: "blockchain1", Address: "addr1"},
		{BlockchainID: "blockchain2", Address: "addr2"},
	})

	var bob account.Account
	if err := accounts.One(db, []byte("bob*iov"), &bob); err != nil {
		t.Fatalf("cannot get bob account: %s", err)
	}
	assert.Equal(t, bob.Domain, "iov")
	assert.Equal(t, bob.Name, "bob")
	assert.Equal(t, bob.Owner, bobCond.Address())
	assert.Equal(t, len(bob.Targets), 0)
}

func TestRewritePreregistrationRecords(t *testing.T) {
	db := store.MemStore()
	migration.MustInitPkg(db, "datamigration", "preregistration", "account")

	ctx := context.Background()
	now := time.Now().UTC()
	ctx = weave.WithBlockTime(ctx, now)

	var (
		adminCond = weave.NewCondition("admin", "test", []byte{1})
		aliceCond = weave.NewCondition("alice", "test", []byte{1})
		bobCond   = weave.NewCondition("bob", "test", []byte{1})
	)

	err := gconf.Save(db, "account", &account.Configuration{
		Metadata:               &weave.Metadata{Schema: 1},
		Owner:                  adminCond.Address(),
		ValidDomain:            `^[a-z]+$`,
		ValidName:              `^[a-z]+$`,
		ValidBlockchainID:      `^[a-z]+$`,
		ValidBlockchainAddress: `^[a-z]+$`,
		DomainRenew:            weave.AsUnixDuration(time.Hour),
	})
	if err != nil {
		t.Fatalf("save account configuration: %s", err)
	}

	records := preregistration.NewRecordBucket()
	_, err = records.Put(db, []byte("alicedomain"), &preregistration.Record{
		Metadata: &weave.Metadata{Schema: 1},
		Domain:   "alicedomain",
		Owner:    aliceCond.Address(),
	})
	if err != nil {
		t.Fatalf("register alice domain: %s", err)
	}
	_, err = records.Put(db, []byte("bobdomain"), &preregistration.Record{
		Metadata: &weave.Metadata{Schema: 1},
		Domain:   "bobdomain",
		Owner:    bobCond.Address(),
	})
	if err != nil {
		t.Fatalf("register bob domain: %s", err)
	}

	if err := rewritePreregistrationRecords(ctx, db); err != nil {
		t.Fatalf("rewrite preregistration records: %s", err)
	}

	domains := account.NewDomainBucket()

	var alice account.Domain
	if err := domains.One(db, []byte("alicedomain"), &alice); err != nil {
		t.Fatalf("cannot get alice account: %s", err)
	}
	assert.Equal(t, alice.Domain, "alicedomain")
	assert.Equal(t, alice.Admin, aliceCond.Address())
	assert.Equal(t, alice.HasSuperuser, true)
	assert.Equal(t, alice.ValidUntil, weave.AsUnixTime(now.Add(time.Hour))) // See Configuration.DomainRenew.

	var bob account.Domain
	if err := domains.One(db, []byte("bobdomain"), &bob); err != nil {
		t.Fatalf("cannot get bob account: %s", err)
	}
	assert.Equal(t, bob.Domain, "bobdomain")
	assert.Equal(t, bob.Admin, bobCond.Address())
	assert.Equal(t, bob.HasSuperuser, true)
	assert.Equal(t, bob.ValidUntil, weave.AsUnixTime(now.Add(time.Hour))) // See Configuration.DomainRenew.
}

func TestRewriteAccountBlockchainIDs(t *testing.T) {
	db := store.MemStore()
	migration.MustInitPkg(db, "datamigration", "account")

	now := time.Now()

	var (
		adminCond = weave.NewCondition("admin", "test", []byte{1})
		aliceCond = weave.NewCondition("alice", "test", []byte{1})
	)

	err := gconf.Save(db, "account", &account.Configuration{
		Metadata:               &weave.Metadata{Schema: 1},
		Owner:                  adminCond.Address(),
		ValidDomain:            `^[a-z]+$`,
		ValidName:              `^[a-z]+$`,
		ValidBlockchainID:      `^[a-z]+$`,
		ValidBlockchainAddress: `^[a-z]+$`,
		DomainRenew:            weave.AsUnixDuration(time.Hour),
	})
	if err != nil {
		t.Fatalf("save account configuration: %s", err)
	}

	domains := account.NewDomainBucket()
	myd := account.Domain{
		Metadata:     &weave.Metadata{Schema: 1},
		Domain:       "myd",
		Admin:        adminCond.Address(),
		ValidUntil:   weave.AsUnixTime(now.Add(time.Hour)),
		HasSuperuser: false,
		MsgFees:      nil,
		AccountRenew: weave.AsUnixDuration(time.Hour),
	}
	if _, err := domains.Put(db, []byte("myd"), &myd); err != nil {
		t.Fatalf("cannot store myd: %s", err)
	}

	accounts := account.NewAccountBucket()

	emptyAccount := account.Account{
		Metadata:   &weave.Metadata{Schema: 1},
		Name:       "",
		Domain:     "myd",
		Owner:      aliceCond.Address(),
		ValidUntil: weave.AsUnixTime(now.Add(time.Hour)),
		Targets: []account.BlockchainAddress{
			{BlockchainID: "unknown", Address: "10"},
			{BlockchainID: "ethereum-eip155-1", Address: "11"},
			{BlockchainID: "iov-mainnet", Address: "12"},
		},
	}
	if _, err := accounts.Put(db, []byte("*myd"), &emptyAccount); err != nil {
		t.Fatalf("cannot save empty account: %s", err)
	}

	aliceAccount := account.Account{
		Metadata:   &weave.Metadata{Schema: 1},
		Name:       "",
		Domain:     "myd",
		Owner:      aliceCond.Address(),
		ValidUntil: weave.AsUnixTime(now.Add(time.Hour)),
		Targets: []account.BlockchainAddress{
			{BlockchainID: "alxchain", Address: "20"},
			{BlockchainID: "ethereum-eip155-1", Address: "21"},
			{BlockchainID: "lisk-ed14889723", Address: "22"},
		},
	}
	if _, err := accounts.Put(db, []byte("alice*myd"), &aliceAccount); err != nil {
		t.Fatalf("cannot save alice account: %s", err)
	}

	if err := rewriteAccountBlockchainIDs(context.Background(), db); err != nil {
		t.Fatalf("rewrite migration: %s", err)
	}

	var acc account.Account

	if err := accounts.One(db, []byte("*myd"), &acc); err != nil {
		t.Fatalf("cannot get empty account: %s", err)
	}
	wantEmptyTargets := []account.BlockchainAddress{
		{BlockchainID: "unknown", Address: "10"},
		{BlockchainID: "eip155:1", Address: "11"},
		{BlockchainID: "cosmos:iov-mainnet", Address: "12"},
	}
	if !reflect.DeepEqual(acc.Targets, wantEmptyTargets) {
		t.Logf("want targets        %+v", wantEmptyTargets)
		t.Fatalf("unexpected targets: %+v", acc.Targets)
	}

	if err := accounts.One(db, []byte("alice*myd"), &acc); err != nil {
		t.Fatalf("cannot get empty account: %s", err)
	}
	wantAliceTargets := []account.BlockchainAddress{
		{BlockchainID: "alxchain", Address: "20"},
		{BlockchainID: "eip155:1", Address: "21"},
		{BlockchainID: "lip9:9ee11e9df416b18b", Address: "22"},
	}
	if !reflect.DeepEqual(acc.Targets, wantAliceTargets) {
		t.Logf("want targets        %+v", wantAliceTargets)
		t.Fatalf("unexpected targets: %+v", acc.Targets)
	}
}
