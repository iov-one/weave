package username_test

import (
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft/username"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeliverIssueTokenMsg(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	auth := helpers.Authenticate(alice)

	kv := store.MemStore()
	bucket := username.NewBucket()

	handler := username.NewIssueHandler(auth, nil, bucket)
	myPubKey := []byte("my Public key")

	// when
	tx := helpers.MockTx(&username.IssueTokenMsg{
		Owner:   alice.Address(),
		Id:      []byte("alice@example.com"),
		Details: username.TokenDetails{[]username.PublicKey{{myPubKey, "myAlgorithm"}}},
	})
	res, err := handler.Deliver(nil, kv, tx)
	// then
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, uint32(0), res.ToABCI().Code)

	// and persisted
	o, err := bucket.Get(kv, []byte("alice@example.com"))
	assert.NoError(t, err)
	assert.NotNil(t, o)
	u, _ := username.AsUsername(o)
	assert.Len(t, u.GetPubKeys(), 1)
	assert.Equal(t, myPubKey, u.GetPubKeys()[0].Data)
}
