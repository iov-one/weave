package sigs

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestUserModel(t *testing.T) {
	kv := store.MemStore()
	migration.MustInitPkg(kv, "sigs")

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

	// Check the sequence several times to ensure that the incrementation
	// works as expected.
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

func TestUserCheckAndIncrementSequence(t *testing.T) {
	cases := map[string]struct {
		User        *UserData
		ExpectedSeq int64
		WantErr     *errors.Error
		WantSeq     int64
	}{
		"a successful first increment": {
			User:        &UserData{Sequence: 0},
			ExpectedSeq: 0,
			WantErr:     nil,
			WantSeq:     1,
		},
		"a successful increment": {
			User:        &UserData{Sequence: 321},
			ExpectedSeq: 321,
			WantErr:     nil,
			WantSeq:     322,
		},
		"the biggest supported sequence value": {
			// This big number is 2^53 - 2
			User:        &UserData{Sequence: 9007199254740990},
			ExpectedSeq: 9007199254740990,
			WantErr:     nil,
			WantSeq:     9007199254740991,
		},
		"a sequence value overflow": {
			// This big number is 2^53 - 1
			User:        &UserData{Sequence: 9007199254740991},
			ExpectedSeq: 9007199254740991,
			WantErr:     errors.ErrOverflow,
			WantSeq:     9007199254740991,
		},
		"a sequence value missmatch": {
			User:        &UserData{Sequence: 333},
			ExpectedSeq: 222,
			WantErr:     ErrInvalidSequence,
			WantSeq:     333,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			err := tc.User.CheckAndIncrementSequence(tc.ExpectedSeq)
			if !tc.WantErr.Is(err) {
				t.Fatalf("unexpected error: %s", err)
			}
			if tc.WantSeq != tc.User.Sequence {
				t.Fatalf("want %v user sequence, got %v", tc.WantSeq, tc.User.Sequence)
			}
		})
	}
}

func TestUserValidation(t *testing.T) {
	cases := map[string]struct {
		User    *UserData
		WantErr *errors.Error
	}{
		"valid": {
			User: &UserData{
				Metadata: &weave.Metadata{Schema: 1},
				Pubkey:   weavetest.NewKey().PublicKey(),
				Sequence: 5,
			},
		},
		"missing metadata": {
			User: &UserData{
				Pubkey:   weavetest.NewKey().PublicKey(),
				Sequence: 5,
			},
			WantErr: errors.ErrMetadata,
		},
		"negative sequence": {
			User: &UserData{
				Metadata: &weave.Metadata{Schema: 1},
				Pubkey:   weavetest.NewKey().PublicKey(),
				Sequence: -5,
			},
			WantErr: ErrInvalidSequence,
		},
		"positive sequence without public key": {
			User: &UserData{
				Metadata: &weave.Metadata{Schema: 1},
				Pubkey:   nil,
				Sequence: 5,
			},
			WantErr: ErrInvalidSequence,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			err := tc.User.Validate()
			if !tc.WantErr.Is(err) {
				t.Fatalf("unexepcted error: %s", err)
			}
		})
	}
}

func TestCannotSetUserPublicKeyTwice(t *testing.T) {
	obj := NewUser(nil)
	if err := obj.Validate(); !errors.ErrEmpty.Is(err) {
		t.Fatalf("unexpected validation error: %s", err)
	}

	pub := weavetest.NewKey().PublicKey()

	AsUser(obj).SetPubkey(pub)
	if err := obj.Validate(); !errors.ErrEmpty.Is(err) {
		t.Fatalf("unexpected validation error: %s", err)
	}

	// Cannot set pubkey a second time.
	assert.Panics(t, func() {
		AsUser(obj).SetPubkey(pub)
	})
}
