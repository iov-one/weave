package account

import (
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestGenesisInitializer(t *testing.T) {
	const genesis = `
	{
		"conf": {
			"account": {
				"valid_name": "^[a-z0-9\\-_.]{3,64}$",
				"valid_domain": "^iov$",
				"valid_blockchain_id": "^[a-z]{3,53}$",
				"valid_blockchain_address": "^[a-z]+$",
				"domain_renew": "2h",
				"owner": "cond:foo/bar/000000000000000001"
			}
		},
		"account": {
			"domains": [
				{
					"admin": "seq:test/alice/1",
					"domain": "first-domain"
				}
			],
			"accounts": [
				{
					"domain": "first-domain",
					"name": "my-account",
					"owner": "seq:test/bob/1"
				}
			]
		}
	}
	`

	var opts weave.Options
	if err := json.Unmarshal([]byte(genesis), &opts); err != nil {
		t.Fatalf("cannot unmarshal genesis: %s", err)
	}

	db := store.MemStore()
	migration.MustInitPkg(db, "account")

	var ini Initializer
	if err := ini.FromGenesis(opts, weave.GenesisParams{}, db); err != nil {
		t.Fatalf("cannot load genesis: %s", err)
	}

	domains := NewDomainBucket()
	var d Domain
	if err := domains.One(db, []byte("first-domain"), &d); err != nil {
		t.Fatalf("cannot get first domain from the database: %s", err)
	}
	assert.Equal(t, d.Admin, weave.NewCondition("test", "alice", weavetest.SequenceID(1)).Address())
	assert.Equal(t, d.Domain, "first-domain")

	accounts := NewAccountBucket()
	var a Account
	if err := accounts.One(db, accountKey("my-account", "first-domain"), &a); err != nil {
		t.Fatalf("cannot get my-account from the database: %s", err)
	}
	assert.Equal(t, a.Owner, weave.NewCondition("test", "bob", weavetest.SequenceID(1)).Address())
	assert.Equal(t, a.Name, "my-account")
	assert.Equal(t, a.Domain, "first-domain")

	var empty Account
	if err := accounts.One(db, accountKey("", "first-domain"), &empty); err != nil {
		t.Fatalf("cannot get empty name account from the database: %s", err)
	}
	assert.Equal(t, empty.Owner, d.Admin)
	assert.Equal(t, empty.Name, "")
	assert.Equal(t, empty.Domain, "first-domain")
}
