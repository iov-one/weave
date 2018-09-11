package humanaddr

import (
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeliverIssueToken(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	auth := helpers.Authenticate(alice)

	kv := store.MemStore()
	bucket := NewBucket()

	handler := NewIssueHandler(auth, nil, bucket)
	myPubKey := []byte("my Public key")

	// when
	tx := helpers.MockTx(&nft.IssueTokenMsg{
		Owner: alice.Address(),
		Id:    []byte("alice@example.com"),
		Details: nft.TokenDetails{Payload: // TODO: revisit. may need custom issue message in this package
		&nft.TokenDetails_HumanAddress{
			&nft.HumanAddressDetails{Account: myPubKey},
		}}})
	res, err := handler.Deliver(nil, kv, tx)
	// then
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, uint32(0), res.ToABCI().Code)
	// todo: id in response?
	// and persisted
	o, err := bucket.Get(kv, []byte("alice@example.com"))
	assert.NoError(t, err)
	u, _ := AsHumanAddress(o)
	assert.Equal(t, myPubKey, u.GetPubKey())
}
