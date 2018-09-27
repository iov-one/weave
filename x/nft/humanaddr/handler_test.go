package humanaddr

import (
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeliverHumanAddressIssueToken(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	auth := helpers.Authenticate(alice)

	kv := store.MemStore()
	bucket := NewBucket()

	handler := NewIssueHandler(auth, nil, bucket)
	myPubKey := []byte("my Public key")

	// when
	tx := helpers.MockTx(&IssueTokenMsg{
		Owner:   alice.Address(),
		Id:      []byte("alice@example.com"),
		Details: TokenDetails{myPubKey},
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
	u, _ := AsHumanAddress(o)
	assert.Equal(t, myPubKey, u.GetPubKey())
}
