package sigs

import (
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestUserModel(t *testing.T) {
	kv := store.MemStore()

	bucket := NewBucket()
	pub := weavetest.NewKey().PublicKey()

	// Not found.
	obj, err := bucket.Get(kv, pub.Address())
	assert.Nil(t, err)
	assert.Nil(t, obj)

	obj, err = bucket.GetOrCreate(kv, pub)
	assert.Nil(t, err)
	if obj == nil {
		t.Fatal("got nil object")
	}
	assert.Nil(t, obj.Validate())

	user := AsUser(obj)
	if user == nil {
		t.Fatal("got nil user")
	}
	assert.Equal(t, int64(0), user.Sequence)

	for i := int64(0); i < 10; i++ {
		if user.CheckAndIncrementSequence(i+10) == nil {
			t.Fatalf("expected the block to be %d", i)
		}
		assert.Nil(t, user.CheckAndIncrementSequence(i))
	}

	err = bucket.Save(kv, obj)
	assert.Nil(t, err)

	obj2, err := bucket.Get(kv, pub.Address())
	assert.Nil(t, err)
	user2 := AsUser(obj2)
	assert.Equal(t, int64(10), user2.Sequence)
	assert.Equal(t, pub, user2.Pubkey)
}

func TestUserValidation(t *testing.T) {
	obj := NewUser(nil)
	if err := obj.Validate(); !errors.ErrEmpty.Is(err) {
		t.Fatalf("unexpected validation error: %s", err)
	}

	pub := weavetest.NewKey().PublicKey()
	AsUser(obj).SetPubkey(pub)
	if err := obj.Validate(); !errors.ErrEmpty.Is(err) {
		t.Fatalf("unexpected validation error: %s", err)
	}

	obj.SetKey(pub.Address())
	assert.Nil(t, obj.Validate())

	// Cannot set pubkey a second time.
	assert.Panics(t, func() {
		AsUser(obj).SetPubkey(pub)
	})

	AsUser(obj).Sequence = -30
	if err := obj.Validate(); !ErrInvalidSequence.Is(err) {
		t.Fatalf("unexpected validation error: %s", err)
	}

	AsUser(obj).Sequence = 17
	assert.Nil(t, obj.Validate())
}
