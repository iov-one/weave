package username

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
		"username": [
			{
				"username": "alice*iov",
				"owner": "seq:test/alice/1",
				"targets": [
					{"blockchain_id": "block_1", "address": "1"},
					{"blockchain_id": "block_2", "address": "2"}
				]
			},
			{
				"username": "charlie*iov",
				"owner": "seq:test/charlie/1",
				"targets": [
					{"blockchain_id": "block_1", "address": "1"}
				]
			}
		]
	}
	`

	var opts weave.Options
	if err := json.Unmarshal([]byte(genesis), &opts); err != nil {
		t.Fatalf("cannot unmarshal genesis: %s", err)
	}

	db := store.MemStore()
	migration.MustInitPkg(db, "username")
	var ini Initializer
	if err := ini.FromGenesis(opts, weave.GenesisParams{}, db); err != nil {
		t.Fatalf("cannot load genesis: %s", err)
	}

	b := NewTokenBucket()
	var alice Token
	if err := b.One(db, []byte("alice*iov"), &alice); err != nil {
		t.Fatalf("cannot get alice from the database: %s", err)
	}
	assert.Equal(t, alice.Owner, weave.NewCondition("test", "alice", weavetest.SequenceID(1)).Address())
	assert.Equal(t, alice.Targets[0].BlockchainID, "block_1")
	assert.Equal(t, alice.Targets[0].Address, "1")
	assert.Equal(t, alice.Targets[1].BlockchainID, "block_2")
	assert.Equal(t, alice.Targets[1].Address, "2")

	var charlie Token
	if err := b.One(db, []byte("charlie*iov"), &charlie); err != nil {
		t.Fatalf("cannot get charlie from the database: %s", err)
	}
	assert.Equal(t, charlie.Owner, weave.NewCondition("test", "charlie", weavetest.SequenceID(1)).Address())
	assert.Equal(t, charlie.Targets[0].BlockchainID, "block_1")
	assert.Equal(t, charlie.Targets[0].Address, "1")
}
