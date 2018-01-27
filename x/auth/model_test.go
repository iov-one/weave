package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"

	crypto "github.com/tendermint/go-crypto"

	"github.com/confio/weave/store"
)

func TestUserModel(t *testing.T) {
	kv := store.MockKVStore()

	key := NewUserKey([]byte("foo"))

	// load fail
	user := GetUser(kv, key)
	assert.Nil(t, user)

	// create
	user = GetOrCreateUser(kv, key)
	assert.NotNil(t, user)
	assert.False(t, user.HasPubKey())
	assert.Equal(t, int64(0), user.Sequence())

	// set
	assert.Error(t, user.CheckAndIncrementSequence(5))
	assert.NoError(t, user.CheckAndIncrementSequence(0))
	assert.Error(t, user.CheckAndIncrementSequence(0))
	assert.NoError(t, user.CheckAndIncrementSequence(1))
	assert.Equal(t, int64(2), user.Sequence())

	// fails with unset pubkey
	assert.Error(t, user.data.Validate())
	assert.Panics(t, func() { user.Save() })

	// set pubkey
	priv := crypto.GenPrivKeyEd25519()
	pub := priv.PubKey()
	user.SetPubKey(pub)
	assert.NoError(t, user.data.Validate())
	// cannot set pubkey a second time....
	assert.Panics(t, func() { user.SetPubKey(pub) })

	// save
	user.Save()

	// load success
	user2 := GetUser(kv, key)
	assert.NotNil(t, user2)
	assert.Equal(t, int64(2), user2.Sequence())
	assert.Equal(t, pub, user2.PubKey())
}
