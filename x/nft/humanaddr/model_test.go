package humanaddr

import (
	"testing"

	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x/nft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueNewHumanAddress(t *testing.T) {
	kv := store.MemStore()
	bucket := NewBucket()
	alice := crypto.GenPrivKeyEd25519().PublicKey().Address()
	myPubKey := []byte("my Public key")
	// when
	o, err := bucket.Create(kv, alice, []byte("alice@example.com"), myPubKey)
	// then
	assert.NoError(t, err)
	assert.NotNil(t, o)
	u, err := AsHumanAddress(o)
	assert.NoError(t, err)
	assert.Equal(t, myPubKey, u.GetPubKey())
}

func TestPersistHumanAddress(t *testing.T) {
	kv := store.MemStore()
	bucket := NewBucket()
	alice := crypto.GenPrivKeyEd25519().PublicKey().Address()
	myPubKey := []byte("my Public key")
	o, _ := bucket.Create(kv, alice, []byte("alice@example.com"), myPubKey)
	// when
	err := bucket.Save(kv, o)
	// then
	assert.NoError(t, err)
	o, err = bucket.Get(kv, []byte("alice@example.com"))
	assert.NoError(t, err)
	u, _ := AsHumanAddress(o)
	assert.Equal(t, myPubKey, u.GetPubKey())
}

func TestTransfer(t *testing.T) {
	alice := crypto.GenPrivKeyEd25519().PublicKey().Address()
	bob := crypto.GenPrivKeyEd25519().PublicKey().Address()

	kv := store.MemStore()
	bucket := NewBucket()
	o, _ := bucket.Create(kv, alice, []byte("alice@example.com"), alice)

	// when
	humanAddress, _ := AsHumanAddress(o)
	err := humanAddress.SetApproval(nft.ActionKind_transferApproval, bob, nil)
	require.NoError(t, err)
	err = humanAddress.Transfer(bob)
	require.NoError(t, err)

	// then
	assert.Equal(t, bob, humanAddress.OwnerAddress())
	assert.Len(t, humanAddress.ApprovalsByAction(nft.ActionKind_transferApproval), 0)
}

func TestRevokeTransfer(t *testing.T) {
	alice := crypto.GenPrivKeyEd25519().PublicKey().Address()
	bob := crypto.GenPrivKeyEd25519().PublicKey().Address()

	kv := store.MemStore()
	bucket := NewBucket()
	o, _ := bucket.Create(kv, alice, []byte("alice@example.com"), alice)

	humanAddress, _ := AsHumanAddress(o)
	err := humanAddress.SetApproval(nft.ActionKind_transferApproval, bob, nil)
	require.NoError(t, err)
	// when
	err = humanAddress.RevokeApproval(nft.ActionKind_transferApproval, bob)
	require.NoError(t, err)
	// then
	assert.Len(t, humanAddress.ApprovalsByAction(nft.ActionKind_transferApproval), 0)
}

func TestUpdatePayload(t *testing.T) {
	alice := crypto.GenPrivKeyEd25519().PublicKey().Address()
	kv := store.MemStore()
	bucket := NewBucket()
	myPubKey := []byte("my Public key")
	o, _ := bucket.Create(kv, alice, []byte("alice@example.com"), myPubKey)
	// when
	u, _ := AsHumanAddress(o)
	myOtherPubKey := []byte("My New Pubkey")
	err := u.SetPubKey(alice, myOtherPubKey)
	// then
	assert.NoError(t, err)
	assert.NotNil(t, o)
	assert.Equal(t, myOtherPubKey, u.GetPubKey())
}

func TestDeleteHumanAddress(t *testing.T) {
	kv := store.MemStore()
	bucket := NewBucket()
	alice := crypto.GenPrivKeyEd25519().PublicKey().Address()
	o, err := bucket.Create(kv, alice, []byte("alice@example.com"), []byte("my Public key"))
	assert.NoError(t, err)
	assert.NoError(t, bucket.Save(kv, o))
	// when
	err = bucket.Delete(kv, []byte("alice@example.com"))
	// then
	assert.NoError(t, err)
	o, err = bucket.Get(kv, []byte("alice@example.com"))
	assert.NoError(t, err)
	assert.Nil(t, o)
}

func TestFindByOwner(t *testing.T) {
	kv := store.MemStore()
	bucket := NewBucket()
	alice := crypto.GenPrivKeyEd25519().PublicKey().Address()
	o1, _ := bucket.Create(kv, alice, []byte("alice1@example.com"), []byte("my Public key"))
	_ = bucket.Save(kv, o1)
	o2, _ := bucket.Create(kv, alice, []byte("alice2@example.com"), []byte("my other key"))
	_ = bucket.Save(kv, o2)
	// when
	result, err := bucket.GetIndexed(kv, OwnerIndexName, []byte(alice))
	// then
	assert.NoError(t, err)
	require.Len(t, result, 2)
	u1, err := AsHumanAddress(result[0])
	assert.NoError(t, err)
	require.NotNil(t, u1)
	assert.Equal(t, []byte("my Public key"), u1.GetPubKey())
	// and
	u2, err := AsHumanAddress(result[1])
	assert.NoError(t, err)
	require.NotNil(t, u2)
	assert.Equal(t, []byte("my other key"), u2.GetPubKey())
}
