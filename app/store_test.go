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
	pubKey := weave.PubKey{
		Type: "test",
		Data: []byte("someKey"),
	}
	pubKey2 := weave.PubKey{
		Type: "test",
		Data: []byte("someKey2"),
	}
	app := NewStoreApp("dummy", iavl.MockCommitStore(), weave.NewQueryRouter(), context.Background())

	t.Run("Diff is equal to output with one update", func(t *testing.T) {
		diff := []weave.ValidatorUpdate{
			{PubKey: pubKey, Power: 10},
		}
		app.AddValChange(diff)
		res := app.EndBlock(abci.RequestEndBlock{})
		assert.Equal(t, weave.ValidatorUpdatesFromABCI(res.ValidatorUpdates).ValidatorUpdates, diff)
	})

	t.Run("Only produce last update to multiple validators", func(t *testing.T) {
		diff := []weave.ValidatorUpdate{
			{PubKey: pubKey, Power: 10},
			{PubKey: pubKey2, Power: 15},
			{PubKey: pubKey, Power: 1},
			{PubKey: pubKey2, Power: 2},
		}

		app.AddValChange(diff)
		res := app.EndBlock(abci.RequestEndBlock{})
		assert.Equal(t, weave.ValidatorUpdatesFromABCI(res.ValidatorUpdates).ValidatorUpdates, diff[2:])
	})

	t.Run("A call with an empty diff does nothing", func(t *testing.T) {
		diff := []weave.ValidatorUpdate{
			{PubKey: pubKey, Power: 10},
			{PubKey: pubKey2, Power: 15},
		}
		app.AddValChange(diff)
		app.AddValChange(make([]weave.ValidatorUpdate, 0))

		res := app.EndBlock(abci.RequestEndBlock{})
		assert.Equal(t, diff, weave.ValidatorUpdatesFromABCI(res.ValidatorUpdates).ValidatorUpdates)
	})
}
