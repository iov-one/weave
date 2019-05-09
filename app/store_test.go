package app

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store/iavl"
	"github.com/iov-one/weave/weavetest/assert"
	abci "github.com/tendermint/tendermint/abci/types"
)

func TestAddValChange(t *testing.T) {
	pubKey := abci.PubKey{
		Type: "test",
		Data: []byte("someKey"),
	}
	pubKey2 := abci.PubKey{
		Type: "test",
		Data: []byte("someKey2"),
	}
	app := NewStoreApp("dummy", iavl.MockCommitStore(), weave.NewQueryRouter(), context.Background())

	t.Run("Diff is equal to output with one update", func(t *testing.T) {
		diff := []abci.ValidatorUpdate{
			{PubKey: pubKey, Power: 10},
		}
		app.AddValChange(diff)
		res := app.EndBlock(abci.RequestEndBlock{})
		assert.Equal(t, res.ValidatorUpdates, diff)
	})

	t.Run("Only produce last update to multiple validators", func(t *testing.T) {
		diff := []abci.ValidatorUpdate{
			{PubKey: pubKey, Power: 10},
			{PubKey: pubKey2, Power: 15},
			{PubKey: pubKey, Power: 1},
			{PubKey: pubKey2, Power: 2},
		}
		app.AddValChange(diff)
		res := app.EndBlock(abci.RequestEndBlock{})
		assert.Equal(t, res.ValidatorUpdates, diff[2:])
	})
}
