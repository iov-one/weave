package bnsd

import (
	"context"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/x/account"
	"github.com/iov-one/weave/cmd/bnsd/x/username"
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
