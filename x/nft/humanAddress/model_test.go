package humanAddress

import (
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x/nft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIssueNewHumanAddress(t *testing.T) {
	kv := store.MemStore()
	bucket := NewBucket()
	myPubKey := []byte("my Public key")
	// when
	o, err := bucket.Create(kv, []byte("alice@example.com"), myPubKey)
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
	myPubKey := []byte("my Public key")
	o, _ := bucket.Create(kv, []byte("alice@example.com"), myPubKey)

	// when
	err := bucket.Save(kv, o)
	// then
	assert.NoError(t, err)
	// and loaded
	o, err = bucket.Get(kv, []byte("alice@example.com"))
	// then
	assert.NoError(t, err)
	u, _ := AsHumanAddress(o)
	assert.Equal(t, myPubKey, u.GetPubKey())
}

func TestTransfer(t *testing.T) {
	alice := crypto.GenPrivKeyEd25519().PublicKey().Address()
	bob := crypto.GenPrivKeyEd25519().PublicKey().Address()

	kv := store.MemStore()
	bucket := NewBucket()
	o, _ := bucket.Create(kv, []byte("alice@example.com"), alice)

	// when
	humanAddress, _ := AsHumanAddress(o)
	err := humanAddress.SetApproval(nft.ActionlKind_transferApproval, bob, nil)
	require.NoError(t, err)
	err = humanAddress.Transfer(bob)
	require.NoError(t, err)
	// then
	assert.Equal(t, bob, humanAddress.OwnerAddress())
	assert.Len(t, humanAddress.Approvals(nft.ActionlKind_transferApproval), 0)
}

func TestRevokeTransfer(t *testing.T) {
	alice := crypto.GenPrivKeyEd25519().PublicKey().Address()
	bob := crypto.GenPrivKeyEd25519().PublicKey().Address()

	kv := store.MemStore()
	bucket := NewBucket()
	o, _ := bucket.Create(kv, []byte("alice@example.com"), alice)

	humanAddress, _ := AsHumanAddress(o)
	err := humanAddress.SetApproval(nft.ActionlKind_transferApproval, bob, nil)
	require.NoError(t, err)
	// when
	err = humanAddress.RevokeApproval(nft.ActionlKind_transferApproval, bob)
	require.NoError(t, err)
	// then
	assert.Len(t, humanAddress.Approvals(nft.ActionlKind_transferApproval), 0)
}

//func TestUpdatePayload(t *testing.T) {
//	kv := store.MemStore()
//	bucket := NewBucket()
//	myPubKey := []byte("my Public key")
//	o, _ := bucket.Create(kv, []byte("alice@example.com"), myPubKey)
//	// when
//	// then
//	assert.NoError(t, err)
//	assert.NotNil(t, o)
//	u, err := AsHumanAddress(o)
//	assert.NoError(t, err)
//	assert.Equal(t, myPubKey, u.GetPubKey())
//
//}
// update
// query
// delete
