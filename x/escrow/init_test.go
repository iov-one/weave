package escrow

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/x/cash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
      "arbiter": "foo/bar/636f6e646974696f6e64617461",
      "recipient": "C30A2424104F542576EF01FECA2FF558F5EAA61A",
      "sender": "0000000000000000000000000000000000000000",
      "timeout": 9223372036854775807
    }
  ]}`

	var opts weave.Options
	require.NoError(t, json.Unmarshal([]byte(genesis), &opts))

	db := store.MemStore()

	// when
	cashCtrl := cash.NewController(cash.NewBucket())
	ini := Initializer{Minter: cashCtrl}
	require.NoError(t, ini.FromGenesis(opts, db))

	// then
	bucket := NewBucket()
	obj, err := bucket.Get(db, weavetest.SequenceID(1))
	require.NoError(t, err)
	require.NotNil(t, obj)
	e, ok := obj.Value().(*Escrow)
	require.True(t, ok)

	require.Len(t, e.Amount, 2)
	assert.Equal(t, coin.Coin{Ticker: "ALX", Whole: 987654321}, *e.Amount[0])
	assert.Equal(t, coin.Coin{Ticker: "IOV", Whole: 123456789}, *e.Amount[1])
	assert.Equal(t, int64(9223372036854775807), e.Timeout)
	assert.Equal(t, "c30a2424104f542576ef01feca2ff558f5eaa61a", hex.EncodeToString(e.Recipient))
	assert.Equal(t, "0000000000000000000000000000000000000000", hex.EncodeToString(e.Sender))

	expArbiter := weave.NewCondition("foo", "bar", []byte("conditiondata"))
	assert.Equal(t, expArbiter, weave.Condition(e.Arbiter))

	balance, err := cashCtrl.Balance(db, Condition(obj.Key()).Address())
	require.NoError(t, err)
	require.Len(t, e.Amount, 2)
	assert.Equal(t, coin.Coin{Ticker: "ALX", Whole: 987654321}, *balance[0])
	assert.Equal(t, coin.Coin{Ticker: "IOV", Whole: 123456789}, *balance[1])
}
