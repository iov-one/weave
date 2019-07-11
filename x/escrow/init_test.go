package escrow

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/cash"
)

func TestGenesisKey(t *testing.T) {
	const genesis = `
{
  "escrow": [
    {
      "amount": [
        {
          "ticker": "ALX",
          "whole": 987654321
        },
        {
          "ticker": "IOV",
          "whole": 123456789
        }
      ],
      "arbiter": "0000000000000000000000000000000000000001",
      "destination": "C30A2424104F542576EF01FECA2FF558F5EAA61A",
      "source": "0000000000000000000000000000000000000000",
      "timeout": "2034-11-10T23:00:00Z"
    }
  ]}`

	var opts weave.Options
	assert.IsErr(t, nil, json.Unmarshal([]byte(genesis), &opts))

	db := store.MemStore()
	migration.MustInitPkg(db, "escrow", "cash")

	// when
	cashCtrl := cash.NewController(cash.NewBucket())
	ini := Initializer{Minter: cashCtrl}
	assert.IsErr(t, nil, ini.FromGenesis(opts, weave.GenesisParams{}, db))

	// then
	bucket := NewBucket()
	obj, err := bucket.Get(db, weavetest.SequenceID(1))
	assert.IsErr(t, nil, err)

	assert.Equal(t, true, obj != nil)
	e, ok := obj.Value().(*Escrow)
	assert.Equal(t, true, ok)

	assert.Equal(t, "c30a2424104f542576ef01feca2ff558f5eaa61a", hex.EncodeToString(e.Destination))
	assert.Equal(t, "0000000000000000000000000000000000000000", hex.EncodeToString(e.Source))
	assert.Equal(t, "0000000000000000000000000000000000000001", hex.EncodeToString(e.Arbiter))

	balance, err := cashCtrl.Balance(db, Condition(obj.Key()).Address())
	assert.IsErr(t, nil, err)
	assert.Equal(t, 2, len(balance))
	assert.Equal(t, coin.Coin{Ticker: "ALX", Whole: 987654321}, *balance[0])
	assert.Equal(t, coin.Coin{Ticker: "IOV", Whole: 123456789}, *balance[1])
}
