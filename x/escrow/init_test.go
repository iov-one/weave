package escrow

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
)

func TestGenesisKey(t *testing.T) {
	const genesis = `
{
  "escrow": [
    {
      "amount": [
        {
          "ticker": "IOV",
          "whole": 123456789
        }
      ],
      "arbiter": "c2lncy9lZDI1NTE5Lyf1+0QFCd+nnsiDoFELyalhTD1EGIiB8MXkAomLS/PJ",
      "recipient": "C30A2424104F542576EF01FECA2FF558F5EAA61A",
      "sender": "0000000000000000000000000000000000000000",
      "timeout": 9223372036854775807
    }
  ]}`

	var opts weave.Options
	if err := json.Unmarshal([]byte(genesis), &opts); err != nil {
		t.Fatalf("cannot unmarshal genesis: %s", err)
	}

	db := store.MemStore()
	var ini Initializer
	if err := ini.FromGenesis(opts, db); err != nil {
		t.Fatalf("cannot load genesis: %s", err)
	}

	bucket := NewBucket()
	obj, err := bucket.Get(db, seq(1))
	if err != nil {
		t.Fatalf("cannot fetch contract information: %s", err)
	}
	if obj == nil {
		t.Fatal("contract information not found")
	}
	_, ok := obj.Value().(*Escrow)
	if !ok {
		t.Errorf("invalid object stored: %T", obj)
	}

	// TODO: check results
}

// seq returns encoded sequence number as implemented in orm/sequence.go
func seq(val int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(val))
	return b
}

func fromHex(t *testing.T, s string) []byte {
	t.Helper()
	raw, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("cannot decode %q hex encoded data: %s", s, err)
	}
	return raw
}
