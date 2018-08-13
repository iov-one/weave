package sigs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/store"
)

func TestUserModel(t *testing.T) {
	kv := store.MemStore()

	bucket := NewBucket()
	pub := crypto.GenPrivKeyEd25519().PublicKey()
	addr := pub.Address()

	// load fail
	obj, err := bucket.Get(kv, addr)
	require.NoError(t, err)
	assert.Nil(t, obj)

	// create
	obj, err = bucket.GetOrCreate(kv, pub)
	require.NoError(t, err)
	assert.NotNil(t, obj)
	assert.NoError(t, obj.Validate())
	user := AsUser(obj)
	assert.NotNil(t, user.PubKey)
	assert.Equal(t, int64(0), user.Sequence)

	// set sequence
	assert.Error(t, user.CheckAndIncrementSequence(5))
	assert.NoError(t, user.CheckAndIncrementSequence(0))
	assert.Error(t, user.CheckAndIncrementSequence(0))
	assert.NoError(t, user.CheckAndIncrementSequence(1))
	assert.Equal(t, int64(2), user.Sequence)

	// save and load
	err = bucket.Save(kv, obj)
	require.NoError(t, err)
	// load success
	obj2, err := bucket.Get(kv, addr)
	require.NoError(t, err)
	assert.NotNil(t, obj2)
	user2 := AsUser(obj2)
	assert.Equal(t, int64(2), user2.Sequence)
	assert.Equal(t, pub, user2.PubKey)
}

func TestUserValidation(t *testing.T) {
	// fails with unset pubkey
	obj := NewUser(nil)
	assert.Error(t, obj.Validate())

	// set pubkey
	pub := crypto.GenPrivKeyEd25519().PublicKey()
	AsUser(obj).SetPubKey(pub)
	assert.Error(t, obj.Validate()) // missing key
	obj.SetKey(pub.Address())
	assert.NoError(t, obj.Validate()) // now complete
	// cannot set pubkey a second time....
	assert.Panics(t, func() { AsUser(obj).SetPubKey(pub) })

	// make sure negative sequence throw error
	AsUser(obj).Sequence = -30
	assert.Error(t, obj.Validate())
	AsUser(obj).Sequence = 17
	assert.NoError(t, obj.Validate())
}
