package multisig

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
)

func TestGenesisKey(t *testing.T) {
	const genesis = `
		{
			"multisig": [
				{
					"sigs": [
						"KFS4NAQmzQQ9aK0Bme3Rs4I0RlE=",
						"NNRgeWsAtyfXCjHo4ynVAiTCC0M=",
						"ucmFiAzJoT51ceK5PuXhZPiU6jU="
					],
					"activation_threshold": 2,
					"admin_threshold": 2
				}
			]
		}
	`

	var opts weave.Options
	if err := json.Unmarshal([]byte(genesis), &opts); err != nil {
		t.Fatalf("cannot unmarshal genesis: %s", err)
	}

	db := store.MemStore()
	var ini Initializer
	if err := ini.FromGenesis(opts, db); err != nil {
		t.Fatalf("cannot load genesis: %s", err)
	}

	bucket := NewContractBucket()
	obj, err := bucket.Get(db, seq(1))
	if err != nil {
		t.Fatalf("cannot fetch contract information: %s", err)
	} else if obj == nil {
		t.Fatal("contract information not found")
	}
	if _, ok := obj.Value().(*Contract); !ok {
		t.Errorf("invalid object stored: %T", obj)
	}
}

// seq returns encoded sequence number as implemented in orm/sequence.go
func seq(val int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(val))
	return b
}
